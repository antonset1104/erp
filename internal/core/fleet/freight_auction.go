package fleet

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CarrierBid represents a binding proposal submitted by a logistics freight forwarder.
type CarrierBid struct {
	BidID                 string  `json:"bid_id"`
	AuctionID             string  `json:"auction_id"`
	CarrierID             string  `json:"carrier_id"`
	CarrierName           string  `json:"carrier_name"`
	ProposedRateUSD       float64 `json:"proposed_rate_usd"`
	EstimatedTransitDays  float64 `json:"estimated_transit_days"`
	CarbonEmissionKgCO2e  float64 `json:"carbon_emission_kg_co2e"`
	HistoricalOnTimeRate  float64 `json:"historical_on_time_rate"` // e.g. 0.96 for 96%
	VesselOrFlightRef     string  `json:"vessel_or_flight_ref"`
	CompositeScore        float64 `json:"composite_score"` // 0.0 to 100.0
	SubmittedAt           string  `json:"submitted_at"`
}

// FreightAuction defines a spot freight multimodal reverse bidding requirement.
type FreightAuction struct {
	AuctionID              string       `json:"auction_id"` // e.g. "AUC-FRT-2026-0812"
	OriginLocation         string       `json:"origin_location"`
	DestinationLocation    string       `json:"destination_location"`
	CargoType              string       `json:"cargo_type"` // e.g. "40ft High-Cube Reefer", "Dry Container", "Heavy Machinery"
	WeightMetricTons       float64      `json:"weight_metric_tons"`
	TargetDeparture        string       `json:"target_departure"`
	MaximumCeilingRateUSD  float64      `json:"maximum_ceiling_rate_usd"`
	BenchmarkCarbonKgCO2e  float64      `json:"benchmark_carbon_kg_co2e"`
	Status                 string       `json:"status"` // "OPEN", "CLOSED_AWARDED", "CANCELLED", "EXPIRED"
	WinningBidID           string       `json:"winning_bid_id,omitempty"`
	AwardedCarrierID       string       `json:"awarded_carrier_id,omitempty"`
	DigitalShippingOrderNo string       `json:"digital_shipping_order_no,omitempty"`
	Bids                   []CarrierBid `json:"bids"`
	CreatedAt              string       `json:"created_at"`
}

// CreateAuctionRequest holds input to publish an enterprise spot bidding event.
type CreateAuctionRequest struct {
	OriginLocation        string  `json:"origin_location"`
	DestinationLocation   string  `json:"destination_location"`
	CargoType             string  `json:"cargo_type"`
	WeightMetricTons      float64 `json:"weight_metric_tons"`
	TargetDeparture       string  `json:"target_departure"`
	MaximumCeilingRateUSD float64 `json:"maximum_ceiling_rate_usd"`
	BenchmarkCarbonKgCO2e float64 `json:"benchmark_carbon_kg_co2e"`
}

// SubmitBidRequest encapsulates a carrier's competitive quote.
type SubmitBidRequest struct {
	AuctionID            string  `json:"auction_id"`
	CarrierID            string  `json:"carrier_id"`
	CarrierName          string  `json:"carrier_name"`
	ProposedRateUSD      float64 `json:"proposed_rate_usd"`
	EstimatedTransitDays float64 `json:"estimated_transit_days"`
	CarbonEmissionKgCO2e float64 `json:"carbon_emission_kg_co2e"`
	HistoricalOnTimeRate float64 `json:"historical_on_time_rate"`
	VesselOrFlightRef    string  `json:"vessel_or_flight_ref"`
}

// AwardAuctionRequest contains the instruction to finalize the auction and issue shipping order.
type AwardAuctionRequest struct {
	AuctionID string `json:"auction_id"`
}

// AuctionAwardResult returns the contractual outcome and economic savings analysis.
type AuctionAwardResult struct {
	AuctionID              string     `json:"auction_id"`
	Status                 string     `json:"status"`
	WinningBid             CarrierBid `json:"winning_bid"`
	DigitalShippingOrderNo string     `json:"digital_shipping_order_no"`
	CeilingRateUSD         float64    `json:"ceiling_rate_usd"`
	AwardedRateUSD         float64    `json:"awarded_rate_usd"`
	TotalSavingsUSD        float64    `json:"total_savings_usd"`
	SavingsPercentage      float64    `json:"savings_percentage"`
	CarbonEfficiencyRating string     `json:"carbon_efficiency_rating"`
	AwardJustification     string     `json:"award_justification"`
}

// FreightAuctionStore manages thread-safe memory storage for freight auctions.
type FreightAuctionStore struct {
	mu       sync.RWMutex
	auctions map[string]*FreightAuction
}

var globalAuctionStore = &FreightAuctionStore{
	auctions: make(map[string]*FreightAuction),
}

func init() {
	// Seed demo freight auction with competitive carrier bids
	demoAuction := &FreightAuction{
		AuctionID:             "AUC-FRT-2026-0812",
		OriginLocation:        "Tanjung Priok Port, Jakarta (IDTPP)",
		DestinationLocation:   "Port of Rotterdam, Netherlands (NLRTM)",
		CargoType:             "40ft High-Cube Cold-Chain Reefer",
		WeightMetricTons:      24.5,
		TargetDeparture:       time.Now().Add(72 * time.Hour).Format("2006-01-02"),
		MaximumCeilingRateUSD: 4800.0,
		BenchmarkCarbonKgCO2e: 3200.0,
		Status:                "OPEN",
		CreatedAt:             time.Now().Format(time.RFC3339),
		Bids:                  make([]CarrierBid, 0),
	}

	globalAuctionStore.auctions[demoAuction.AuctionID] = demoAuction

	// Pre-seed 3 competitive carrier bids
	_, _ = SubmitBidLogic(SubmitBidRequest{
		AuctionID:            demoAuction.AuctionID,
		CarrierID:            "CAR-MAERSK-01",
		CarrierName:          "Maersk Ocean Line",
		ProposedRateUSD:      4250.0,
		EstimatedTransitDays: 22.0,
		CarbonEmissionKgCO2e: 2450.0,
		HistoricalOnTimeRate: 0.95,
		VesselOrFlightRef:    "Vessel MSC Oscar v.044W",
	})
	_, _ = SubmitBidLogic(SubmitBidRequest{
		AuctionID:            demoAuction.AuctionID,
		CarrierID:            "CAR-HAPAG-02",
		CarrierName:          "Hapag-Lloyd Green Express",
		ProposedRateUSD:      3900.0,
		EstimatedTransitDays: 24.0,
		CarbonEmissionKgCO2e: 2100.0,
		HistoricalOnTimeRate: 0.92,
		VesselOrFlightRef:    "Vessel Berlin Express v.102E",
	})
	_, _ = SubmitBidLogic(SubmitBidRequest{
		AuctionID:            demoAuction.AuctionID,
		CarrierID:            "CAR-ONE-03",
		CarrierName:          "Ocean Network Express (ONE)",
		ProposedRateUSD:      4400.0,
		EstimatedTransitDays: 20.0,
		CarbonEmissionKgCO2e: 2800.0,
		HistoricalOnTimeRate: 0.88,
		VesselOrFlightRef:    "Vessel ONE Apus v.009",
	})
}

// calculateCompositeScore computes multi-factor score:
// Rate (40%), Transit Speed (25%), Carbon Efficiency (20%), Carrier Reliability (15%).
func calculateCompositeScore(bid CarrierBid, ceilingUSD float64, benchmarkCarbon float64) float64 {
	// Rate score: up to 40 pts
	rateScore := 0.0
	if ceilingUSD > 0 && bid.ProposedRateUSD <= ceilingUSD {
		savingsRatio := (ceilingUSD - bid.ProposedRateUSD) / ceilingUSD
		rateScore = 20.0 + (savingsRatio * 20.0) // 20 base for being under ceiling + up to 20 for savings
	}

	// Transit score: up to 25 pts (baseline 30 days)
	transitScore := 0.0
	if bid.EstimatedTransitDays > 0 {
		transitScore = math.Max(0.0, (35.0-bid.EstimatedTransitDays)/35.0) * 25.0
	}

	// Carbon score: up to 20 pts
	carbonScore := 0.0
	if benchmarkCarbon > 0 && bid.CarbonEmissionKgCO2e <= benchmarkCarbon {
		carbonScore = ((benchmarkCarbon - bid.CarbonEmissionKgCO2e) / benchmarkCarbon) * 20.0
	}

	// Reliability score: up to 15 pts
	reliabilityScore := bid.HistoricalOnTimeRate * 15.0

	composite := rateScore + transitScore + carbonScore + reliabilityScore
	return math.Round(composite*100) / 100
}

// CreateAuctionLogic opens a new spot freight reverse bidding event.
func CreateAuctionLogic(req CreateAuctionRequest) (*FreightAuction, error) {
	globalAuctionStore.mu.Lock()
	defer globalAuctionStore.mu.Unlock()

	if req.OriginLocation == "" || req.DestinationLocation == "" {
		return nil, fmt.Errorf("origin and destination locations are mandatory")
	}
	if req.MaximumCeilingRateUSD <= 0 {
		req.MaximumCeilingRateUSD = 5000.0
	}
	if req.BenchmarkCarbonKgCO2e <= 0 {
		req.BenchmarkCarbonKgCO2e = 3000.0
	}

	auctionID := fmt.Sprintf("AUC-FRT-2026-%04d", len(globalAuctionStore.auctions)+813)
	auction := &FreightAuction{
		AuctionID:             auctionID,
		OriginLocation:        req.OriginLocation,
		DestinationLocation:   req.DestinationLocation,
		CargoType:             req.CargoType,
		WeightMetricTons:      req.WeightMetricTons,
		TargetDeparture:       req.TargetDeparture,
		MaximumCeilingRateUSD: req.MaximumCeilingRateUSD,
		BenchmarkCarbonKgCO2e: req.BenchmarkCarbonKgCO2e,
		Status:                "OPEN",
		CreatedAt:             time.Now().Format(time.RFC3339),
		Bids:                  make([]CarrierBid, 0),
	}

	globalAuctionStore.auctions[auctionID] = auction
	return auction, nil
}

// SubmitBidLogic adds or updates a carrier's bid and recalculates relative rankings.
func SubmitBidLogic(req SubmitBidRequest) (*CarrierBid, error) {
	globalAuctionStore.mu.Lock()
	defer globalAuctionStore.mu.Unlock()

	auction, exists := globalAuctionStore.auctions[req.AuctionID]
	if !exists {
		return nil, fmt.Errorf("freight auction '%s' not found", req.AuctionID)
	}
	if auction.Status != "OPEN" {
		return nil, fmt.Errorf("auction '%s' is not open for bidding (current status: %s)", req.AuctionID, auction.Status)
	}
	if req.ProposedRateUSD > auction.MaximumCeilingRateUSD {
		return nil, fmt.Errorf("proposed rate $%.2f exceeds ceiling cap $%.2f", req.ProposedRateUSD, auction.MaximumCeilingRateUSD)
	}

	bidID := fmt.Sprintf("BID-%s-%03d", req.CarrierID, len(auction.Bids)+1)
	bid := CarrierBid{
		BidID:                bidID,
		AuctionID:            req.AuctionID,
		CarrierID:            req.CarrierID,
		CarrierName:          req.CarrierName,
		ProposedRateUSD:      req.ProposedRateUSD,
		EstimatedTransitDays: req.EstimatedTransitDays,
		CarbonEmissionKgCO2e: req.CarbonEmissionKgCO2e,
		HistoricalOnTimeRate: req.HistoricalOnTimeRate,
		VesselOrFlightRef:    req.VesselOrFlightRef,
		SubmittedAt:          time.Now().Format(time.RFC3339),
	}

	bid.CompositeScore = calculateCompositeScore(bid, auction.MaximumCeilingRateUSD, auction.BenchmarkCarbonKgCO2e)

	// Replace existing bid if carrier already submitted, else append
	found := false
	for i, b := range auction.Bids {
		if b.CarrierID == req.CarrierID {
			auction.Bids[i] = bid
			found = true
			break
		}
	}
	if !found {
		auction.Bids = append(auction.Bids, bid)
	}

	// Sort bids descending by composite score
	sort.Slice(auction.Bids, func(i, j int) bool {
		return auction.Bids[i].CompositeScore > auction.Bids[j].CompositeScore
	})

	return &bid, nil
}

// AwardAuctionLogic selects the top-scoring carrier bid, creates DSO, and closes the auction.
func AwardAuctionLogic(req AwardAuctionRequest) (*AuctionAwardResult, error) {
	globalAuctionStore.mu.Lock()
	defer globalAuctionStore.mu.Unlock()

	auction, exists := globalAuctionStore.auctions[req.AuctionID]
	if !exists {
		return nil, fmt.Errorf("freight auction '%s' not found", req.AuctionID)
	}
	if auction.Status != "OPEN" {
		return nil, fmt.Errorf("auction '%s' cannot be awarded (status: %s)", req.AuctionID, auction.Status)
	}
	if len(auction.Bids) == 0 {
		return nil, fmt.Errorf("cannot award auction with 0 submitted bids")
	}

	winningBid := auction.Bids[0] // Highest composite score
	dsoNumber := fmt.Sprintf("DSO-2026-%s-%04d", winningBid.CarrierID, time.Now().UnixNano()%10000)

	savingsUSD := auction.MaximumCeilingRateUSD - winningBid.ProposedRateUSD
	savingsPct := (savingsUSD / auction.MaximumCeilingRateUSD) * 100.0

	carbonRating := "B"
	if winningBid.CarbonEmissionKgCO2e <= auction.BenchmarkCarbonKgCO2e*0.75 {
		carbonRating = "A+"
	} else if winningBid.CarbonEmissionKgCO2e <= auction.BenchmarkCarbonKgCO2e*0.90 {
		carbonRating = "A"
	}

	auction.Status = "CLOSED_AWARDED"
	auction.WinningBidID = winningBid.BidID
	auction.AwardedCarrierID = winningBid.CarrierID
	auction.DigitalShippingOrderNo = dsoNumber

	justification := fmt.Sprintf("Awarded to %s with best composite score (%.2f/100). Yields $%.2f (%.1f%%) cost reduction and %s carbon efficiency.",
		winningBid.CarrierName, winningBid.CompositeScore, savingsUSD, savingsPct, carbonRating)

	return &AuctionAwardResult{
		AuctionID:              auction.AuctionID,
		Status:                 auction.Status,
		WinningBid:             winningBid,
		DigitalShippingOrderNo: dsoNumber,
		CeilingRateUSD:         auction.MaximumCeilingRateUSD,
		AwardedRateUSD:         winningBid.ProposedRateUSD,
		TotalSavingsUSD:        savingsUSD,
		SavingsPercentage:      savingsPct,
		CarbonEfficiencyRating: carbonRating,
		AwardJustification:     justification,
	}, nil
}

// RegisterFreightAuctionRoutes mounts spot freight reverse auction API endpoints.
func RegisterFreightAuctionRoutes(r *gin.RouterGroup) {
	group := r.Group("/fleet/freight-auction")
	{
		group.POST("/create", func(c *gin.Context) {
			var req CreateAuctionRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid create auction payload: " + err.Error()})
				return
			}
			res, err := CreateAuctionLogic(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, res)
		})

		group.POST("/submit-bid", func(c *gin.Context) {
			var req SubmitBidRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid submit bid payload: " + err.Error()})
				return
			}
			res, err := SubmitBidLogic(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.POST("/award", func(c *gin.Context) {
			var req AwardAuctionRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid award auction payload: " + err.Error()})
				return
			}
			res, err := AwardAuctionLogic(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/list", func(c *gin.Context) {
			globalAuctionStore.mu.RLock()
			defer globalAuctionStore.mu.RUnlock()

			list := make([]*FreightAuction, 0, len(globalAuctionStore.auctions))
			for _, a := range globalAuctionStore.auctions {
				list = append(list, a)
			}
			c.JSON(http.StatusOK, gin.H{
				"total_auctions": len(list),
				"auctions":       list,
			})
		})
	}
}
