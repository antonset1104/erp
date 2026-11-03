package expense

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ExpenseClaimItem struct {
	ClaimID        string  `json:"claim_id"`
	EmployeeID     uint    `json:"employee_id"`
	EmployeeName   string  `json:"employee_name"`
	Department     string  `json:"department"`
	ClaimDate      string  `json:"claim_date"` // YYYY-MM-DD
	Category       string  `json:"category"`   // MEALS, TRAVEL, LODGING, ENTERTAINMENT, OFFICE_SUPPLIES
	MerchantName   string  `json:"merchant_name"`
	Amount         float64 `json:"amount"`
	ReceiptHash    string  `json:"receipt_hash"` // Digital perceptual hash or invoice ref
	Description    string  `json:"description"`
	IsWeekendClaim bool    `json:"is_weekend_claim"`
}

type FraudViolation struct {
	ViolationType string  `json:"violation_type"` // SPLIT_TRANSACTION_SMURFING, DUPLICATE_RECEIPT, NON_WORKING_DAY, PROHIBITED_MERCHANT, LUXURY_POLICY_BREACH
	Severity      string  `json:"severity"`       // LOW, MEDIUM, HIGH, CRITICAL
	RiskScore     float64 `json:"risk_score"`
	Description   string  `json:"description"`
	Evidence      string  `json:"evidence"`
}

type ExpenseAuditResult struct {
	ClaimID           string           `json:"claim_id"`
	EmployeeName      string           `json:"employee_name"`
	MerchantName      string           `json:"merchant_name"`
	Amount            float64          `json:"amount"`
	OverallRiskScore  float64          `json:"overall_risk_score"` // 0 - 100
	AuditStatus       string           `json:"audit_status"`       // APPROVED_CLEAN, SUSPICIOUS_REVIEW, FRAUD_HOLD
	ViolationsCount   int              `json:"violations_count"`
	Violations        []FraudViolation `json:"violations"`
	PolicyAction      string           `json:"policy_action"`
	AuditedAt         string           `json:"audited_at"`
}

type ExpenseAuditBatchResponse struct {
	TotalClaimsAudited int                  `json:"total_claims_audited"`
	CleanCount         int                  `json:"clean_count"`
	ReviewRequired     int                  `json:"review_required"`
	FraudHoldCount     int                  `json:"fraud_hold_count"`
	TotalFlaggedAmount float64              `json:"total_flagged_amount"`
	AuditResults       []ExpenseAuditResult `json:"audit_results"`
	ExecutiveAdvice    string               `json:"executive_advice"`
	AuditedAt          string               `json:"audited_at"`
}

var expenseStore = struct {
	sync.RWMutex
	flaggedClaims []ExpenseAuditResult
}{
	flaggedClaims: make([]ExpenseAuditResult, 0),
}

// EvaluateExpenseClaims audits expense claims against anti-fraud policies
func EvaluateExpenseClaims(claims []ExpenseClaimItem) *ExpenseAuditBatchResponse {
	if len(claims) == 0 {
		// Provide default representative test claims
		claims = []ExpenseClaimItem{
			{
				ClaimID:        "EXP-2026-901",
				EmployeeID:     1,
				EmployeeName:   "Bambang Pratama",
				Department:     "Regional Sales",
				ClaimDate:      "2026-11-01",
				Category:       "ENTERTAINMENT",
				MerchantName:   "Grand Karaoke & Lounge Suites",
				Amount:         2850000,
				ReceiptHash:    "HASH-RC-998811",
				Description:    "Jamuan makan malam klien VIP PT Global",
				IsWeekendClaim: true,
			},
			{
				ClaimID:        "EXP-2026-902",
				EmployeeID:     1,
				EmployeeName:   "Bambang Pratama",
				Department:     "Regional Sales",
				ClaimDate:      "2026-11-01",
				Category:       "MEALS",
				MerchantName:   "Resto Nusantara Senayan",
				Amount:         980000, // Just below 1.0M approval ceiling
				ReceiptHash:    "HASH-RC-772211",
				Description:    "Snack meeting klien bagian 1",
				IsWeekendClaim: true,
			},
			{
				ClaimID:        "EXP-2026-903",
				EmployeeID:     1,
				EmployeeName:   "Bambang Pratama",
				Department:     "Regional Sales",
				ClaimDate:      "2026-11-01",
				Category:       "MEALS",
				MerchantName:   "Resto Nusantara Senayan",
				Amount:         950000, // Second split below 1.0M ceiling
				ReceiptHash:    "HASH-RC-772212",
				Description:    "Snack meeting klien bagian 2",
				IsWeekendClaim: true,
			},
			{
				ClaimID:        "EXP-2026-904",
				EmployeeID:     2,
				EmployeeName:   "Siti Rahma",
				Department:     "Finance & Accounting",
				ClaimDate:      "2026-11-02",
				Category:       "TRAVEL",
				MerchantName:   "Bluebird Taxi Jakarta",
				Amount:         185000,
				ReceiptHash:    "HASH-TX-009988",
				Description:    "Transport dinas ke kantor pajak KPP Pratama",
				IsWeekendClaim: false,
			},
			{
				ClaimID:        "EXP-2026-905",
				EmployeeID:     3,
				EmployeeName:   "Dodi Firmansyah",
				Department:     "Operations",
				ClaimDate:      "2026-11-02",
				Category:       "TRAVEL",
				MerchantName:   "Bluebird Taxi Jakarta",
				Amount:         185000,
				ReceiptHash:    "HASH-TX-009988", // Exact duplicate receipt hash!
				Description:    "Klaim taksi bandara",
				IsWeekendClaim: false,
			},
		}
	}

	var results []ExpenseAuditResult
	cleanCount := 0
	reviewCount := 0
	holdCount := 0
	flaggedTotal := 0.0

	// Check duplicates by receipt hash and merchant/amount pairs
	hashMap := make(map[string][]string) // hash -> claimIDs
	employeeDayAmounts := make(map[string]float64) // empID:date -> total amount
	employeeDayCounts := make(map[string]int)

	for _, c := range claims {
		if c.ReceiptHash != "" {
			hashMap[c.ReceiptHash] = append(hashMap[c.ReceiptHash], c.ClaimID)
		}
		key := fmt.Sprintf("%d:%s", c.EmployeeID, c.ClaimDate)
		employeeDayAmounts[key] += c.Amount
		employeeDayCounts[key]++
	}

	for _, c := range claims {
		var violations []FraudViolation
		maxScore := 0.0

		// Rule 1: Duplicate receipt claim across claims
		if len(hashMap[c.ReceiptHash]) > 1 {
			v := FraudViolation{
				ViolationType: "DUPLICATE_RECEIPT",
				Severity:      "CRITICAL",
				RiskScore:     95.0,
				Description:   "Nomor struk/hash digital identik telah diajukan pada klaim lain.",
				Evidence:      fmt.Sprintf("Struk '%s' digunakan berulang kali pada klaim: %v", c.ReceiptHash, hashMap[c.ReceiptHash]),
			}
			violations = append(violations, v)
			maxScore = math.Max(maxScore, v.RiskScore)
		}

		// Rule 2: Smurfing / Split claim below approval limit
		key := fmt.Sprintf("%d:%s", c.EmployeeID, c.ClaimDate)
		if employeeDayCounts[key] >= 2 && c.Amount >= 800000 && c.Amount < 1000000 {
			v := FraudViolation{
				ViolationType: "SPLIT_TRANSACTION_SMURFING",
				Severity:      "HIGH",
				RiskScore:     82.0,
				Description:   "Indikasi pemecahan transaksi (smurfing) untuk menghindari batas persetujuan manajerial Rp 1.000.000.",
				Evidence:      fmt.Sprintf("Karyawan mengajukan %d klaim pada tanggal yang sama (%s) dengan total kumulatif Rp %.0f.", employeeDayCounts[key], c.ClaimDate, employeeDayAmounts[key]),
			}
			violations = append(violations, v)
			maxScore = math.Max(maxScore, v.RiskScore)
		}

		// Rule 3: Prohibited luxury or entertainment venue
		lowerDesc := strings.ToLower(c.Description + " " + c.MerchantName)
		if strings.Contains(lowerDesc, "karaoke") || strings.Contains(lowerDesc, "lounge") || strings.Contains(lowerDesc, "spa") || strings.Contains(lowerDesc, "wine") {
			v := FraudViolation{
				ViolationType: "PROHIBITED_MERCHANT",
				Severity:      "CRITICAL",
				RiskScore:     92.0,
				Description:   "Merchant/aktivitas tergolong hiburan terlarang dan melanggar Buku Kebijakan Etika Beban Usaha.",
				Evidence:      fmt.Sprintf("Terdeteksi kata kunci terlarang pada merchant: '%s'", c.MerchantName),
			}
			violations = append(violations, v)
			maxScore = math.Max(maxScore, v.RiskScore)
		}

		// Rule 4: Weekend claim without authorization
		if c.IsWeekendClaim {
			v := FraudViolation{
				ViolationType: "NON_WORKING_DAY",
				Severity:      "MEDIUM",
				RiskScore:     45.0,
				Description:   "Klaim biaya dilakukan pada akhir pekan / hari libur tanpa lampiran surat tugas lembur.",
				Evidence:      fmt.Sprintf("Tanggal klaim %s jatuh pada hari libur.", c.ClaimDate),
			}
			violations = append(violations, v)
			maxScore = math.Max(maxScore, v.RiskScore)
		}

		status := "APPROVED_CLEAN"
		action := "LULUS AUDIT: Klaim memenuhi seluruh parameter kepatuhan SOP beban usaha GoERP."

		if maxScore >= 75.0 {
			status = "FRAUD_HOLD"
			action = "TAHAN PENCAIRAN: Terindikasi pelanggaran kritis. Wajib audit investigasi internal sebelum disetujui direksi."
			holdCount++
			flaggedTotal += c.Amount
		} else if maxScore >= 40.0 {
			status = "SUSPICIOUS_REVIEW"
			action = "TINJAU ULANG: Minta konfirmasi tambahan dan otorisasi manajer divisi sebelum pembayaran diproses."
			reviewCount++
			flaggedTotal += c.Amount
		} else {
			cleanCount++
		}

		res := ExpenseAuditResult{
			ClaimID:          c.ClaimID,
			EmployeeName:     c.EmployeeName,
			MerchantName:     c.MerchantName,
			Amount:           c.Amount,
			OverallRiskScore: maxScore,
			AuditStatus:      status,
			ViolationsCount:  len(violations),
			Violations:       violations,
			PolicyAction:     action,
			AuditedAt:        time.Now().Format("2006-01-02 15:04:05"),
		}
		results = append(results, res)
	}

	expenseStore.Lock()
	expenseStore.flaggedClaims = results
	expenseStore.Unlock()

	advice := fmt.Sprintf("AI Expense Auditor: Dari %d klaim yang diperiksa, %d klaim berhasil DITAHAN (FRAUD_HOLD) dan %d klaim membutuhkan review lanjutan. Menyelamatkan potensi kebocoran kas sebesar IDR %.0f.",
		len(claims), holdCount, reviewCount, flaggedTotal)

	return &ExpenseAuditBatchResponse{
		TotalClaimsAudited: len(claims),
		CleanCount:         cleanCount,
		ReviewRequired:     reviewCount,
		FraudHoldCount:     holdCount,
		TotalFlaggedAmount: flaggedTotal,
		AuditResults:       results,
		ExecutiveAdvice:    advice,
		AuditedAt:          time.Now().Format("2006-01-02 15:04:05"),
	}
}

// RegisterExpenseAuditRoutes mounts HTTP endpoints for Expense Fraud Auditing
func RegisterExpenseAuditRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	grp := router.Group("/expense/audit")
	grp.Use(authMiddleware)
	{
		grp.POST("/evaluate", func(c *gin.Context) {
			var claims []ExpenseClaimItem
			_ = c.ShouldBindJSON(&claims)
			resp := EvaluateExpenseClaims(claims)
			c.JSON(http.StatusOK, resp)
		})

		grp.GET("/flagged", func(c *gin.Context) {
			expenseStore.RLock()
			defer expenseStore.RUnlock()
			c.JSON(http.StatusOK, gin.H{
				"flagged_claims": expenseStore.flaggedClaims,
				"total":          len(expenseStore.flaggedClaims),
			})
		})
	}
}
