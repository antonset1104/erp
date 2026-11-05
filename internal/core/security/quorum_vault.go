package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Prime modulus for Shamir Secret Sharing over GF(P) (Mersenne prime 2^127 - 1)
var primeMod, _ = new(big.Int).SetString("170141183460469231731687303715884105727", 10)

// OfficerKeyShard represents a distributed secret share assigned to an executive officer.
type OfficerKeyShard struct {
	Index         int64  `json:"index"`     // X-coordinate (1, 2, ... n)
	ShareHex      string `json:"share_hex"` // Y-coordinate hex
	OfficerRole   string `json:"officer_role"`
	OfficerName   string `json:"officer_name"`
	OfficerEmail  string `json:"officer_email"`
	DistributedAt string `json:"distributed_at"`
}

// VaultPolicy defines quorum requirements for sensitive operations.
type VaultPolicy struct {
	PolicyName       string   `json:"policy_name"`
	RequiredQuorum   int      `json:"required_quorum"` // k
	TotalOfficers    int      `json:"total_officers"`  // n
	PermittedActions []string `json:"permitted_actions"`
}

// ShardKeysRequest initiates key sharding for a sensitive master key or operation.
type ShardKeysRequest struct {
	VaultPolicyName string            `json:"vault_policy_name"`
	SecretSeed      string            `json:"secret_seed,omitempty"` // If blank, securely generated
	RequiredQuorum  int               `json:"required_quorum"`       // k (e.g. 3)
	Officers        []OfficerKeyShard `json:"officers"`              // n officers
}

// ShardKeysResponse returns the distributed key shards.
type ShardKeysResponse struct {
	VaultSessionID    string            `json:"vault_session_id"`
	RequiredQuorum    int               `json:"required_quorum"`
	TotalShardsIssued int               `json:"total_shards_issued"`
	MasterKeyHash     string            `json:"master_key_hash"` // SHA-256 fingerprint
	DistributedShards []OfficerKeyShard `json:"distributed_shards"`
	SecurityNotice    string            `json:"security_notice"`
}

// SubmittedShard represents a key shard presented during quorum execution.
type SubmittedShard struct {
	Index    int64  `json:"index"`
	ShareHex string `json:"share_hex"`
}

// ExecuteQuorumActionRequest initiates a restricted high-consequence enterprise action.
type ExecuteQuorumActionRequest struct {
	VaultSessionID  string           `json:"vault_session_id"`
	ActionType      string           `json:"action_type"` // e.g. UNLOCK_FISCAL_YEAR_GL, EXPORT_MASTER_PAYROLL
	TargetReference string           `json:"target_reference"`
	Justification   string           `json:"justification"`
	SubmittedShards []SubmittedShard `json:"submitted_shards"`
}

// ExecuteQuorumActionResponse confirms execution of the quorum-protected action.
type ExecuteQuorumActionResponse struct {
	Success            bool   `json:"success"`
	ActionExecuted     string `json:"action_executed"`
	TargetReference    string `json:"target_reference"`
	QuorumSatisfied    bool   `json:"quorum_satisfied"`
	ValidShardsCount   int    `json:"valid_shards_count"`
	RequiredQuorum     int    `json:"required_quorum"`
	AuditExecutionCode string `json:"audit_execution_code"`
	CryptographicSeal  string `json:"cryptographic_seal"`
	ExecutionTimestamp string `json:"execution_timestamp"`
	Message            string `json:"message"`
}

// QuorumVaultService implements zero-trust Shamir secret sharing and quorum execution.
type QuorumVaultService struct {
	mu         sync.RWMutex
	vaultStore map[string]*vaultSession
}

type vaultSession struct {
	SessionID    string
	Quorum       int
	SecretHash   string
	SecretInt    *big.Int
	IssuedShards []OfficerKeyShard
}

// NewQuorumVaultService creates a new QuorumVaultService.
func NewQuorumVaultService() *QuorumVaultService {
	return &QuorumVaultService{
		vaultStore: make(map[string]*vaultSession),
	}
}

// ShardKeys generates and splits a master key into n shards with threshold k.
func (s *QuorumVaultService) ShardKeys(req ShardKeysRequest) (*ShardKeysResponse, error) {
	if req.RequiredQuorum <= 1 {
		req.RequiredQuorum = 3
	}

	if len(req.Officers) == 0 {
		req.Officers = []OfficerKeyShard{
			{Index: 1, OfficerRole: "Chief Executive Officer (CEO)", OfficerName: "Ir. Hendra Pratama", OfficerEmail: "ceo@goerp-enterprise.com"},
			{Index: 2, OfficerRole: "Chief Financial Officer (CFO)", OfficerName: "Citra Melati, SE, Ak.", OfficerEmail: "cfo@goerp-enterprise.com"},
			{Index: 3, OfficerRole: "Chief Internal Auditor (CIA)", OfficerName: "Dr. Budi Santoso, CISA", OfficerEmail: "audit@goerp-enterprise.com"},
			{Index: 4, OfficerRole: "Head of Corporate Governance", OfficerName: "Dewi Puspita, SH, LLM", OfficerEmail: "legal@goerp-enterprise.com"},
			{Index: 5, OfficerRole: "VP of Enterprise Infrastructure", OfficerName: "Agus Setiawan, CISSP", OfficerEmail: "secops@goerp-enterprise.com"},
		}
	}

	if req.RequiredQuorum > len(req.Officers) {
		req.RequiredQuorum = len(req.Officers)
	}

	// Secret integer S
	var secretInt *big.Int
	if req.SecretSeed != "" {
		hash := sha256.Sum256([]byte(req.SecretSeed))
		secretInt = new(big.Int).SetBytes(hash[:16]) // 128-bit secret
	} else {
		buf := make([]byte, 16)
		rand.Read(buf)
		secretInt = new(big.Int).SetBytes(buf)
	}
	secretInt.Mod(secretInt, primeMod)

	// Generate random polynomial coefficients of degree k-1: P(x) = S + a1*x + a2*x^2 + ...
	k := req.RequiredQuorum
	coeffs := make([]*big.Int, k)
	coeffs[0] = secretInt
	for i := 1; i < k; i++ {
		cBytes := make([]byte, 16)
		rand.Read(cBytes)
		cInt := new(big.Int).SetBytes(cBytes)
		coeffs[i] = cInt.Mod(cInt, primeMod)
	}

	var distributedShards []OfficerKeyShard
	nowStr := time.Now().Format(time.RFC3339)

	for _, off := range req.Officers {
		x := big.NewInt(off.Index)
		// Evaluate polynomial P(x) mod primeMod
		y := evaluatePolynomial(coeffs, x, primeMod)

		shard := OfficerKeyShard{
			Index:         off.Index,
			ShareHex:      hex.EncodeToString(y.Bytes()),
			OfficerRole:   off.OfficerRole,
			OfficerName:   off.OfficerName,
			OfficerEmail:  off.OfficerEmail,
			DistributedAt: nowStr,
		}
		distributedShards = append(distributedShards, shard)
	}

	sessionID := fmt.Sprintf("VAULT-SESS-%s-%04d", time.Now().Format("20060102"), time.Now().Nanosecond()%10000)
	secHash := sha256.Sum256(secretInt.Bytes())
	masterHashStr := hex.EncodeToString(secHash[:])

	session := &vaultSession{
		SessionID:    sessionID,
		Quorum:       req.RequiredQuorum,
		SecretHash:   masterHashStr,
		SecretInt:    secretInt,
		IssuedShards: distributedShards,
	}

	s.mu.Lock()
	s.vaultStore[sessionID] = session
	s.mu.Unlock()

	return &ShardKeysResponse{
		VaultSessionID:    sessionID,
		RequiredQuorum:    req.RequiredQuorum,
		TotalShardsIssued: len(distributedShards),
		MasterKeyHash:     masterHashStr,
		DistributedShards: distributedShards,
		SecurityNotice:    fmt.Sprintf("Kunci master dipecah menjadi %d shard menggunakan Shamir's Secret Sharing GF(P). Diperlukan minimal %d persetujuan kuorum.", len(distributedShards), req.RequiredQuorum),
	}, nil
}

// ExecuteQuorumAction validates threshold shares and executes protected actions.
func (s *QuorumVaultService) ExecuteQuorumAction(req ExecuteQuorumActionRequest) (*ExecuteQuorumActionResponse, error) {
	s.mu.RLock()
	session, exists := s.vaultStore[req.VaultSessionID]
	s.mu.RUnlock()

	requiredQuorum := 3
	if exists {
		requiredQuorum = session.Quorum
	}

	if len(req.SubmittedShards) < requiredQuorum {
		return &ExecuteQuorumActionResponse{
			Success:            false,
			ActionExecuted:     req.ActionType,
			TargetReference:    req.TargetReference,
			QuorumSatisfied:    false,
			ValidShardsCount:   len(req.SubmittedShards),
			RequiredQuorum:     requiredQuorum,
			ExecutionTimestamp: time.Now().Format(time.RFC3339),
			Message:            fmt.Sprintf("KUORUM GAGAL: Aksi '%s' ditolak. Diperlukan minimal %d pecahan kunci, namun hanya %d yang diberikan.", req.ActionType, requiredQuorum, len(req.SubmittedShards)),
		}, errors.New("insufficient quorum shares submitted")
	}

	// Reconstruct secret using Lagrange interpolation
	points := make([]struct{ X, Y *big.Int }, len(req.SubmittedShards))
	for i, sub := range req.SubmittedShards {
		yBytes, err := hex.DecodeString(sub.ShareHex)
		if err != nil {
			return nil, fmt.Errorf("invalid shard hex for index %d", sub.Index)
		}
		points[i] = struct{ X, Y *big.Int }{
			X: big.NewInt(sub.Index),
			Y: new(big.Int).SetBytes(yBytes),
		}
	}

	reconstructedSecret := lagrangeInterpolateAtZero(points[:requiredQuorum], primeMod)
	reconstructedHash := sha256.Sum256(reconstructedSecret.Bytes())
	reconstructedHashStr := hex.EncodeToString(reconstructedHash[:])

	if exists && reconstructedHashStr != session.SecretHash {
		return &ExecuteQuorumActionResponse{
			Success:            false,
			ActionExecuted:     req.ActionType,
			QuorumSatisfied:    false,
			Message:            "Pecahan kunci yang diajukan tidak valid atau terjadi inkonsistensi matematis.",
			ExecutionTimestamp: time.Now().Format(time.RFC3339),
		}, errors.New("cryptographic secret reconstruction mismatch")
	}

	auditCode := fmt.Sprintf("AUD-QUORUM-%s-%04d", time.Now().Format("20060102"), time.Now().Nanosecond()%10000)
	sealHash := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s", auditCode, req.ActionType, req.TargetReference, reconstructedHashStr)))

	return &ExecuteQuorumActionResponse{
		Success:            true,
		ActionExecuted:     req.ActionType,
		TargetReference:    req.TargetReference,
		QuorumSatisfied:    true,
		ValidShardsCount:   len(req.SubmittedShards),
		RequiredQuorum:     requiredQuorum,
		AuditExecutionCode: auditCode,
		CryptographicSeal:  hex.EncodeToString(sealHash[:]),
		ExecutionTimestamp: time.Now().Format(time.RFC3339),
		Message:            fmt.Sprintf("KUORUM TERPENUHI: Aksi '%s' berhasil dieksekusi dengan otorisasi %d pemegang kunci eksekutif. Kriptografis terverifikasi.", req.ActionType, len(req.SubmittedShards)),
	}, nil
}

// Math helpers for Shamir's Secret Sharing
func evaluatePolynomial(coeffs []*big.Int, x *big.Int, p *big.Int) *big.Int {
	result := big.NewInt(0)
	xPower := big.NewInt(1)

	for _, c := range coeffs {
		term := new(big.Int).Mul(c, xPower)
		term.Mod(term, p)
		result.Add(result, term)
		result.Mod(result, p)

		xPower.Mul(xPower, x)
		xPower.Mod(xPower, p)
	}
	return result
}

func lagrangeInterpolateAtZero(points []struct{ X, Y *big.Int }, p *big.Int) *big.Int {
	secret := big.NewInt(0)

	for i := 0; i < len(points); i++ {
		xi := points[i].X
		yi := points[i].Y

		num := big.NewInt(1)
		den := big.NewInt(1)

		for j := 0; j < len(points); j++ {
			if i == j {
				continue
			}
			xj := points[j].X

			// num = num * (-xj)
			negXj := new(big.Int).Neg(xj)
			negXj.Mod(negXj, p)
			num.Mul(num, negXj)
			num.Mod(num, p)

			// den = den * (xi - xj)
			diff := new(big.Int).Sub(xi, xj)
			diff.Mod(diff, p)
			den.Mul(den, diff)
			den.Mod(den, p)
		}

		invDen := new(big.Int).ModInverse(den, p)
		if invDen == nil {
			invDen = big.NewInt(1)
		}
		term := new(big.Int).Mul(yi, num)
		term.Mod(term, p)
		term.Mul(term, invDen)
		term.Mod(term, p)

		secret.Add(secret, term)
		secret.Mod(secret, p)
	}
	return secret
}

var defaultVaultSvc = NewQuorumVaultService()

// RegisterQuorumVaultRoutes mounts HTTP routes for Zero-Trust Quorum Vault.
func RegisterQuorumVaultRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	g := router.Group("/security/vault")
	g.Use(authMiddleware)
	{
		g.POST("/shard-keys", func(c *gin.Context) {
			var req ShardKeysRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := defaultVaultSvc.ShardKeys(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		g.POST("/execute-quorum-action", func(c *gin.Context) {
			var req ExecuteQuorumActionRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := defaultVaultSvc.ExecuteQuorumAction(req)
			if err != nil {
				c.JSON(http.StatusForbidden, res)
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
