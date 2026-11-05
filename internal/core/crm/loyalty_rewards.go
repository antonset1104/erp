package crm

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// LoyaltyMember tracks a customer's rewards tier, points balance, and IFRS 15 deferred revenue liability.
type LoyaltyMember struct {
	MemberID                        string    `json:"member_id"`
	CustomerName                    string    `json:"customer_name"`
	TierLevel                       string    `json:"tier_level"` // "BRONZE", "SILVER", "GOLD", "PLATINUM", "VIP_ENTERPRISE"
	TotalPointsBalance              int       `json:"total_points_balance"`
	LifetimePointsEarned            int       `json:"lifetime_points_earned"`
	TotalSpentIDR                   float64   `json:"total_spent_idr"`
	ContractLiabilityDeferredRevIDR float64   `json:"contract_liability_deferred_rev_idr"` // IFRS 15 / PSAK 72
	LastTransactionDate             string    `json:"last_transaction_date"`
	CreatedAt                       time.Time `json:"created_at"`
}

// EarnPointsRequest represents an omnichannel sale yielding rewards points.
type EarnPointsRequest struct {
	MemberID             string  `json:"member_id"`
	CustomerName         string  `json:"customer_name"`
	TransactionAmountIDR float64 `json:"transaction_amount_idr"`
	SalesOrderID         string  `json:"sales_order_id"`
	Channel              string  `json:"channel"` // "POS_RETAIL", "ECOMMERCE_WEB", "B2B_PORTAL"
}

// EarnPointsResult provides the points calculation, tier progression, and IFRS 15 accounting impact.
type EarnPointsResult struct {
	MemberID                 string  `json:"member_id"`
	CustomerName             string  `json:"customer_name"`
	TierLevel                string  `json:"tier_level"`
	PointsEarned             int     `json:"points_earned"`
	NewTotalPointsBalance    int     `json:"new_total_points_balance"`
	TierUpgraded             bool    `json:"tier_upgraded"`
	PreviousTier             string  `json:"previous_tier,omitempty"`
	IFRS15DeferredRevenueIDR float64 `json:"ifrs15_deferred_revenue_idr"` // Deferred from sales invoice
	Message                  string  `json:"message"`
}

// RedeemRewardRequest defines points burn for vouchers or merchandise.
type RedeemRewardRequest struct {
	MemberID          string `json:"member_id"`
	PointsToRedeem    int    `json:"points_to_redeem"`
	RewardCatalogItem string `json:"reward_catalog_item"` // e.g. "VOUCHER-DISCOUNT-50K", "ANNUAL-FREE-FREIGHT"
}

// RedemptionResult details the redeemed voucher and released IFRS 15 contract liability.
type RedemptionResult struct {
	RedemptionCode           string  `json:"redemption_code"`
	MemberID                 string  `json:"member_id"`
	RewardCatalogItem        string  `json:"reward_catalog_item"`
	PointsDeducted           int     `json:"points_deducted"`
	RemainingPointsBalance   int     `json:"remaining_points_balance"`
	ReleasedIFRS15RevenueIDR float64 `json:"released_ifrs15_revenue_idr"` // Reclassified from Liability to Earned Revenue
	RedemptionStatus         string  `json:"redemption_status"`
	RedeemedAt               string  `json:"redeemed_at"`
}

var (
	loyaltyMutex sync.RWMutex
	loyaltyStore = make(map[string]*LoyaltyMember)
)

func init() {
	// Seed demo loyalty member
	demoMember := &LoyaltyMember{
		MemberID:                        "MBR-2026-VIP-001",
		CustomerName:                    "PT Nusantara Retail Perkasa",
		TierLevel:                       "GOLD",
		TotalPointsBalance:              8500,
		LifetimePointsEarned:            12500,
		TotalSpentIDR:                   125000000.0,
		ContractLiabilityDeferredRevIDR: 722500.0, // 8500 pts * Rp 85
		LastTransactionDate:             "2026-10-25",
		CreatedAt:                       time.Now().AddDate(0, -6, 0),
	}
	loyaltyStore[demoMember.MemberID] = demoMember
}

// getTierMultiplier returns bonus points scaling per tier.
func getTierMultiplier(tier string) float64 {
	switch tier {
	case "SILVER":
		return 1.25
	case "GOLD":
		return 1.50
	case "PLATINUM":
		return 2.00
	case "VIP_ENTERPRISE":
		return 2.50
	default:
		return 1.00 // Bronze
	}
}

// evaluateTier checks lifetime points thresholds for automatic tier promotion.
func evaluateTier(lifetimePoints int) string {
	switch {
	case lifetimePoints >= 50000:
		return "VIP_ENTERPRISE"
	case lifetimePoints >= 15000:
		return "PLATINUM"
	case lifetimePoints >= 5000:
		return "GOLD"
	case lifetimePoints >= 1000:
		return "SILVER"
	default:
		return "BRONZE"
	}
}

// EarnLoyaltyPointsLogic calculates earned points, evaluates tier upgrades, and defers IFRS 15 liability.
func EarnLoyaltyPointsLogic(req EarnPointsRequest) (*EarnPointsResult, error) {
	loyaltyMutex.Lock()
	defer loyaltyMutex.Unlock()

	if req.MemberID == "" {
		req.MemberID = fmt.Sprintf("MBR-%s-%03d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	}
	if req.CustomerName == "" {
		req.CustomerName = "Valued Enterprise Customer"
	}

	mbr, exists := loyaltyStore[req.MemberID]
	if !exists {
		mbr = &LoyaltyMember{
			MemberID:                        req.MemberID,
			CustomerName:                    req.CustomerName,
			TierLevel:                       "BRONZE",
			TotalPointsBalance:              0,
			LifetimePointsEarned:            0,
			TotalSpentIDR:                   0.0,
			ContractLiabilityDeferredRevIDR: 0.0,
			CreatedAt:                       time.Now(),
		}
		loyaltyStore[req.MemberID] = mbr
	}

	// Base rate: 1 point per Rp 10,000 spent
	basePoints := int(req.TransactionAmountIDR / 10000.0)
	multiplier := getTierMultiplier(mbr.TierLevel)
	pointsEarned := int(float64(basePoints) * multiplier)
	if pointsEarned <= 0 {
		pointsEarned = 1
	}

	// IFRS 15 / PSAK 72: Point standalone selling price = Rp 100, Breakage = 15% -> Rp 85 net liability deferred
	deferredRevenueToAdd := float64(pointsEarned) * 85.0

	prevTier := mbr.TierLevel
	mbr.TotalPointsBalance += pointsEarned
	mbr.LifetimePointsEarned += pointsEarned
	mbr.TotalSpentIDR += req.TransactionAmountIDR
	mbr.ContractLiabilityDeferredRevIDR += deferredRevenueToAdd
	mbr.LastTransactionDate = time.Now().Format("2006-01-02")

	newTier := evaluateTier(mbr.LifetimePointsEarned)
	tierUpgraded := newTier != prevTier
	mbr.TierLevel = newTier

	msg := fmt.Sprintf("Customer earned %d points. Total balance: %d points.", pointsEarned, mbr.TotalPointsBalance)
	if tierUpgraded {
		msg += fmt.Sprintf(" CONGRATULATIONS: Tier upgraded from %s to %s!", prevTier, newTier)
	}

	return &EarnPointsResult{
		MemberID:                 mbr.MemberID,
		CustomerName:             mbr.CustomerName,
		TierLevel:                mbr.TierLevel,
		PointsEarned:             pointsEarned,
		NewTotalPointsBalance:    mbr.TotalPointsBalance,
		TierUpgraded:             tierUpgraded,
		PreviousTier:             prevTier,
		IFRS15DeferredRevenueIDR: deferredRevenueToAdd,
		Message:                  msg,
	}, nil
}

// RedeemLoyaltyRewardLogic burns member points and releases IFRS 15 deferred revenue liability.
func RedeemLoyaltyRewardLogic(req RedeemRewardRequest) (*RedemptionResult, error) {
	loyaltyMutex.Lock()
	defer loyaltyMutex.Unlock()

	mbr, exists := loyaltyStore[req.MemberID]
	if !exists {
		return nil, fmt.Errorf("loyalty member '%s' not found", req.MemberID)
	}

	if req.PointsToRedeem <= 0 {
		return nil, fmt.Errorf("points to redeem must be greater than zero")
	}

	if mbr.TotalPointsBalance < req.PointsToRedeem {
		return nil, fmt.Errorf("insufficient points balance (%d available, %d requested)", mbr.TotalPointsBalance, req.PointsToRedeem)
	}

	mbr.TotalPointsBalance -= req.PointsToRedeem

	// IFRS 15 Liability release: Rp 85 per point reclassified from Contract Liability to Earned Revenue
	releasedRev := float64(req.PointsToRedeem) * 85.0
	mbr.ContractLiabilityDeferredRevIDR -= releasedRev
	if mbr.ContractLiabilityDeferredRevIDR < 0 {
		mbr.ContractLiabilityDeferredRevIDR = 0
	}

	redemptionCode := fmt.Sprintf("RDM-VCH-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)

	return &RedemptionResult{
		RedemptionCode:           redemptionCode,
		MemberID:                 mbr.MemberID,
		RewardCatalogItem:        req.RewardCatalogItem,
		PointsDeducted:           req.PointsToRedeem,
		RemainingPointsBalance:   mbr.TotalPointsBalance,
		ReleasedIFRS15RevenueIDR: releasedRev,
		RedemptionStatus:         "VOUCHER_ACTIVATED",
		RedeemedAt:               time.Now().Format(time.RFC3339),
	}, nil
}

// GetLoyaltyMemberProfileLogic retrieves real-time member profile and liability balance.
func GetLoyaltyMemberProfileLogic(memberID string) (*LoyaltyMember, error) {
	loyaltyMutex.RLock()
	defer loyaltyMutex.RUnlock()

	mbr, exists := loyaltyStore[memberID]
	if !exists {
		return nil, fmt.Errorf("member '%s' not found", memberID)
	}
	return mbr, nil
}

// RegisterLoyaltyRoutes registers customer loyalty rewards and IFRS 15 contract liability routes.
func RegisterLoyaltyRoutes(r *gin.RouterGroup) {
	group := r.Group("/crm/loyalty")
	{
		group.POST("/earn-points", func(c *gin.Context) {
			var req EarnPointsRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
				return
			}
			res, err := EarnLoyaltyPointsLogic(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.POST("/redeem-reward", func(c *gin.Context) {
			var req RedeemRewardRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
				return
			}
			res, err := RedeemLoyaltyRewardLogic(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/member/:member_id", func(c *gin.Context) {
			mid := c.Param("member_id")
			res, err := GetLoyaltyMemberProfileLogic(mid)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
