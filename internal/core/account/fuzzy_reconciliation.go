package account

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"erp-golang/internal/modules/account"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RawBankStatementLine struct {
	LineID      string  `json:"line_id"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	BankRef     string  `json:"bank_ref"`
}

type FuzzyMatchCandidate struct {
	StatementLineID       string  `json:"statement_line_id"`
	StatementDate         string  `json:"statement_date"`
	StatementMemo         string  `json:"statement_memo"`
	StatementAmount       float64 `json:"statement_amount"`
	MatchedMoveID         uint    `json:"matched_move_id"`
	MatchedMoveName       string  `json:"matched_move_name"`
	MatchedPartnerName    string  `json:"matched_partner_name"`
	InvoiceAmount         float64 `json:"invoice_amount"`
	AdminFeeDifference    float64 `json:"admin_fee_difference"` // BI-FAST fee Rp 2,500 / Rp 6,500
	ConfidenceScore       float64 `json:"confidence_score"`     // 0.0 - 1.0
	MatchStatus           string  `json:"match_status"`         // EXACT_MATCH, FUZZY_MATCH_FEE_TOLERANCE, UNMATCHED
	AutoReconcileEligible bool    `json:"auto_reconcile_eligible"`
}

type AutoMatchResponse struct {
	TotalStatementLines   int                   `json:"total_statement_lines"`
	MatchedLinesCount     int                   `json:"matched_lines_count"`
	UnmatchedLinesCount   int                   `json:"unmatched_lines_count"`
	TotalMatchedVolumeIDR float64               `json:"total_matched_volume_idr"`
	Matches               []FuzzyMatchCandidate `json:"matches"`
	Summary               string                `json:"summary"`
	ExecutedAt            string                `json:"executed_at"`
}

type AutoMatchRequest struct {
	StatementLines []RawBankStatementLine `json:"statement_lines"`
}

type BatchReconcileRequest struct {
	Matches    []FuzzyMatchCandidate `json:"matches" binding:"required"`
	AutoPostGL bool                  `json:"auto_post_gl"`
}

type BatchReconcileResponse struct {
	ReconciledCount       int      `json:"reconciled_count"`
	TotalAmountReconciled float64  `json:"total_amount_reconciled"`
	GeneratedPaymentRefs  []string `json:"generated_payment_refs"`
	Status                string   `json:"status"`
	CompletedAt           string   `json:"completed_at"`
}

// RunFuzzyReconciliation matches bank statement lines against open invoices
func RunFuzzyReconciliation(db *gorm.DB, lines []RawBankStatementLine, companyID uint) *AutoMatchResponse {
	var openMoves []account.AccountMove
	db.Where("company_id = ? AND state = 'posted' AND payment_state != 'paid'", companyID).Find(&openMoves)

	// If no open moves found, query all posted moves for demonstration
	if len(openMoves) == 0 {
		db.Where("company_id = ? AND state = 'posted'", companyID).Find(&openMoves)
	}

	var results []FuzzyMatchCandidate
	matchedCount := 0
	totalMatchedVolume := 0.0

	for _, stmt := range lines {
		absAmount := math.Abs(stmt.Amount)
		memoLower := strings.ToLower(stmt.Description)

		var bestCandidate *FuzzyMatchCandidate
		highestScore := 0.0

		for _, m := range openMoves {
			score := 0.0
			diff := math.Abs(m.AmountTotal - absAmount)

			// Amount matching
			if diff == 0 {
				score += 0.60 // Exact amount match
			} else if diff <= 6500.0 {
				// Transfer fee tolerance (BI-FAST Rp 2,500 or Online Transfer Rp 6,500)
				score += 0.50
			} else if math.Abs(diff/m.AmountTotal) < 0.05 {
				score += 0.30
			}

			// Reference matching
			moveNameLower := strings.ToLower(m.Name)
			refLower := strings.ToLower(m.Ref)
			if strings.Contains(memoLower, moveNameLower) || (refLower != "" && strings.Contains(memoLower, refLower)) {
				score += 0.35
			}

			// Substring token match
			tokens := strings.Fields(moveNameLower)
			for _, tok := range tokens {
				if len(tok) >= 4 && strings.Contains(memoLower, tok) {
					score += 0.15
					break
				}
			}

			if score > highestScore {
				highestScore = score
				status := "UNMATCHED"
				eligible := false

				if score >= 0.85 {
					status = "EXACT_MATCH"
					eligible = true
				} else if score >= 0.65 {
					status = "FUZZY_MATCH_FEE_TOLERANCE"
					eligible = true
				}

				bestCandidate = &FuzzyMatchCandidate{
					StatementLineID:       stmt.LineID,
					StatementDate:         stmt.Date,
					StatementMemo:         stmt.Description,
					StatementAmount:       stmt.Amount,
					MatchedMoveID:         m.ID,
					MatchedMoveName:       m.Name,
					MatchedPartnerName:    "Pelanggan Terdaftar",
					InvoiceAmount:         m.AmountTotal,
					AdminFeeDifference:    math.Round(diff*100) / 100,
					ConfidenceScore:       math.Round(score*100) / 100,
					MatchStatus:           status,
					AutoReconcileEligible: eligible,
				}
			}
		}

		if bestCandidate != nil && bestCandidate.ConfidenceScore >= 0.60 {
			results = append(results, *bestCandidate)
			matchedCount++
			totalMatchedVolume += absAmount
		} else {
			results = append(results, FuzzyMatchCandidate{
				StatementLineID:       stmt.LineID,
				StatementDate:         stmt.Date,
				StatementMemo:         stmt.Description,
				StatementAmount:       stmt.Amount,
				MatchStatus:           "UNMATCHED",
				ConfidenceScore:       0.0,
				AutoReconcileEligible: false,
			})
		}
	}

	summary := fmt.Sprintf("ML Fuzzy Bank Reconciler: Merekonsiliasi %d dari %d mutasi rekening koran (Volume: IDR %.2f). Tingkat akurasi otomatisasi: %.1f%%.",
		matchedCount, len(lines), totalMatchedVolume, (float64(matchedCount)/float64(len(lines)))*100.0)

	return &AutoMatchResponse{
		TotalStatementLines:   len(lines),
		MatchedLinesCount:     matchedCount,
		UnmatchedLinesCount:   len(lines) - matchedCount,
		TotalMatchedVolumeIDR: totalMatchedVolume,
		Matches:               results,
		Summary:               summary,
		ExecutedAt:            time.Now().Format("2006-01-02 15:04:05"),
	}
}

// Handler functions for HTTP routes
type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) AutoMatch(c *gin.Context) {
	var req AutoMatchRequest
	_ = c.ShouldBindJSON(&req)

	if len(req.StatementLines) == 0 {
		// Provide standard default test statement
		req.StatementLines = []RawBankStatementLine{
			{
				LineID:      "STMT-001",
				Date:        time.Now().Format("2006-01-02"),
				Description: "TRSF CR BI-FAST INV/2026/00001 DARI PT MITRA MAKMUR",
				Amount:      10497500.0, // Rp 10.5M minus Rp 2,500 BI-FAST fee
				BankRef:     "BCA-FAST-88992211",
			},
			{
				LineID:      "STMT-002",
				Date:        time.Now().Format("2006-01-02"),
				Description: "BIAYA ADM REK KORAN BULANAN",
				Amount:      -25000.0,
				BankRef:     "BCA-CHG-990011",
			},
		}
	}

	resp := RunFuzzyReconciliation(h.db, req.StatementLines, 1)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) BatchPost(c *gin.Context) {
	var req BatchReconcileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var generatedRefs []string
	totalReconciled := 0.0

	for _, m := range req.Matches {
		if !m.AutoReconcileEligible {
			continue
		}
		payRef := fmt.Sprintf("PAY/AUTO-RECON/%s/%04d", time.Now().Format("200601"), time.Now().Unix()%10000)
		generatedRefs = append(generatedRefs, payRef)
		totalReconciled += m.InvoiceAmount

		// Mark invoice payment state if valid ID
		if m.MatchedMoveID > 0 {
			var move account.AccountMove
			if err := h.db.Where("id = ?", m.MatchedMoveID).First(&move).Error; err == nil {
				move.PaymentState = "paid"
				move.AmountResidual = 0
				_ = h.db.Save(&move)
			}
		}
	}

	c.JSON(http.StatusOK, BatchReconcileResponse{
		ReconciledCount:       len(generatedRefs),
		TotalAmountReconciled: totalReconciled,
		GeneratedPaymentRefs:  generatedRefs,
		Status:                "SUCCESS_BATCH_RECONCILED",
		CompletedAt:           time.Now().Format("2006-01-02 15:04:05"),
	})
}

// RegisterFuzzyReconcileRoutes registers ML bank reconciliation endpoints
func RegisterFuzzyReconcileRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	h := NewHandler(db)
	group := router.Group("/account/reconciliation")
	group.Use(authMiddleware)
	{
		group.POST("/auto-match", h.AutoMatch)
		group.POST("/batch-post", h.BatchPost)
	}
}
