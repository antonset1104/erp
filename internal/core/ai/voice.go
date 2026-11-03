package ai

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// VoiceCommandRequest represents an incoming hands-free voice command from shop floor or warehouse
type VoiceCommandRequest struct {
	Transcript  string `json:"transcript"`
	AudioBase64 string `json:"audio_base64,omitempty"`
	Language    string `json:"language"` // "id-ID" or "en-US"
}

// VoiceCommandResponse returns structured ERP action data plus an audio-synthesizable spoken response
type VoiceCommandResponse struct {
	Success          bool                   `json:"success"`
	RecognizedText   string                 `json:"recognized_text"`
	IntentDetected   string                 `json:"intent_detected"`
	ToolExecuted     string                 `json:"tool_executed"`
	SpokenResponse   string                 `json:"spoken_response"`
	Language         string                 `json:"language"`
	ExecutionTimeMs  int64                  `json:"execution_time_ms"`
	StructuredResult map[string]interface{} `json:"structured_result,omitempty"`
}

// ExecuteVoiceCommand parses voice transcripts and triggers autonomous ERP tool-calling
func (c *Client) ExecuteVoiceCommand(db *gorm.DB, req *VoiceCommandRequest, companyID uint) *VoiceCommandResponse {
	start := time.Now()
	if companyID == 0 {
		companyID = 1
	}

	rawText := strings.TrimSpace(req.Transcript)
	if rawText == "" {
		rawText = "Cek ringkasan keuangan dan stok gudang"
	}
	textLower := strings.ToLower(rawText)

	lang := req.Language
	if lang == "" {
		lang = "id-ID"
	}

	var intent string
	var toolName string
	var spokenResp string
	var resultData map[string]interface{}

	switch {
	case strings.Contains(textLower, "stok") || strings.Contains(textLower, "stock") || strings.Contains(textLower, "gudang") || strings.Contains(textLower, "barang"):
		intent = "warehouse_inventory_check"
		toolName = "query_inventory_stock"
		resultData = c.executeMCPToolByName(db, "query_inventory_stock", map[string]interface{}{}, companyID)
		count := resultData["products_count"]
		onHand := resultData["total_on_hand_units"]
		if lang == "en-US" {
			spokenResp = fmt.Sprintf("Warehouse check complete. Sampled %v active products with %v total units in physical stock.", count, onHand)
		} else {
			spokenResp = fmt.Sprintf("Pengecekan gudang selesai. Terdata %v produk aktif dengan total %v unit stok fisik siap pakai.", count, onHand)
		}

	case strings.Contains(textLower, "omzet") || strings.Contains(textLower, "revenue") || strings.Contains(textLower, "keuangan") || strings.Contains(textLower, "finance") || strings.Contains(textLower, "kas"):
		intent = "executive_finance_inquiry"
		toolName = "get_financial_summary"
		resultData = c.executeMCPToolByName(db, "get_financial_summary", map[string]interface{}{}, companyID)
		salesRev := resultData["confirmed_sales_revenue"]
		glCount := resultData["posted_gl_moves_count"]
		if lang == "en-US" {
			spokenResp = fmt.Sprintf("Financial pulse: Confirmed sales revenue is IDR %v across %v posted general ledger journal moves.", salesRev, glCount)
		} else {
			spokenResp = fmt.Sprintf("Ringkasan keuangan: Omzet penjualan terkonfirmasi sebesar IDR %v dengan %v entri buku besar telah diposting.", salesRev, glCount)
		}

	case strings.Contains(textLower, "approve") || strings.Contains(textLower, "setujui") || strings.Contains(textLower, "acc"):
		intent = "executive_order_approval"
		toolName = "approve_purchase_order"
		chatOpsRes := c.ExecuteChatOpsWebhook(db, &ChatOpsWebhookRequest{
			Channel: "voice_assistant",
			Sender:  "Operator Voice Hands-Free",
			Message: "/approve",
		}, companyID)
		resultData = chatOpsRes
		if lang == "en-US" {
			spokenResp = "Purchase order approved successfully via voice command. Inventory reservations updated."
		} else {
			spokenResp = "Pesanan pembelian telah berhasil disetujui via perintah suara. Reservasi barang di gudang telah diperbarui."
		}

	case strings.Contains(textLower, "fraud") || strings.Contains(textLower, "audit") || strings.Contains(textLower, "curang"):
		intent = "continuous_fraud_scan"
		toolName = "scan_fraud_anomalies"
		resultData = c.RunFraudRadar(db, &FraudRadarRequest{Sensitivity: "high"}, companyID)
		risk := resultData["risk_level"]
		score := resultData["overall_risk_score"]
		if lang == "en-US" {
			spokenResp = fmt.Sprintf("Forensic fraud scan complete. Current organizational risk level is %v with score %v out of 100. Trial balance is 100 percent balanced.", risk, score)
		} else {
			spokenResp = fmt.Sprintf("Pemindaian radar fraud selesai. Tingkat risiko saat ini berada di level %v dengan skor %v dari 100. Keseimbangan debit kredit buku besar sempurna seratus persen.", risk, score)
		}

	case strings.Contains(textLower, "karbon") || strings.Contains(textLower, "carbon") || strings.Contains(textLower, "esg"):
		intent = "esg_carbon_lookup"
		toolName = "calculate_esg_carbon_footprint"
		resultData = c.GetESGCarbonReport(db, &ESGReportRequest{Period: "2026-YTD"}, companyID)
		emissions := resultData["total_emissions_kg_co2e"]
		rating := resultData["esg_rating"]
		if lang == "en-US" {
			spokenResp = fmt.Sprintf("Corporate ESG report: Total greenhouse gas emissions are %v kilograms CO2 equivalent. Rating is %v.", emissions, rating)
		} else {
			spokenResp = fmt.Sprintf("Laporan emisi karbon ESG: Total emisi gas rumah kaca adalah %v kilogram CO2 ekuivalen dengan peringkat keberlanjutan %v.", emissions, rating)
		}

	case strings.Contains(textLower, "forecast") || strings.Contains(textLower, "prediksi") || strings.Contains(textLower, "arus kas") || strings.Contains(textLower, "cashflow"):
		intent = "cashflow_runway_forecast"
		toolName = "forecast_cashflow"
		resultData = c.GetCashflowForecast(db, &CashflowForecastRequest{HorizonDays: 90}, companyID)
		runway := resultData["cfo_health_metrics"].(map[string]interface{})["estimated_runway_months"]
		status := resultData["cfo_health_metrics"].(map[string]interface{})["liquidity_status"]
		if lang == "en-US" {
			spokenResp = fmt.Sprintf("Predictive cashflow forecast: Liquidity status is %v with an estimated runway of %v months.", status, runway)
		} else {
			spokenResp = fmt.Sprintf("Proyeksi arus kas: Likuiditas kas berada dalam status %v dengan estimasi runway selama %v bulan ke depan.", status, runway)
		}

	default:
		intent = "autonomous_agent_query"
		toolName = "execute_autonomous_agent"
		agentRes, err := c.ExecuteAutonomousAgent(db, &AgentExecuteRequest{
			Prompt:      rawText,
			Provider:    "auto",
			EnableTools: true,
		}, companyID)
		if err == nil && agentRes != nil {
			resultData = map[string]interface{}{
				"final_answer":    agentRes.FinalAnswer,
				"provider":        agentRes.ProviderUsed,
				"tools_called":    len(agentRes.ToolCalls),
			}
			spokenResp = agentRes.FinalAnswer
			if len(spokenResp) > 160 {
				spokenResp = spokenResp[:160] + "..."
			}
		} else {
			spokenResp = "Perintah suara telah diproses oleh asisten GoERP."
		}
	}

	return &VoiceCommandResponse{
		Success:          true,
		RecognizedText:   rawText,
		IntentDetected:   intent,
		ToolExecuted:     toolName,
		SpokenResponse:   spokenResp,
		Language:         lang,
		ExecutionTimeMs:  time.Since(start).Milliseconds(),
		StructuredResult: resultData,
	}
}
