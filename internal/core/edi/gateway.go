package edi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-golang/internal/core/middleware"
	"erp-golang/internal/modules/sale"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EDILineItem struct {
	LineNumber  int     `json:"line_number"`
	ProductCode string  `json:"product_code"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitPrice   float64 `json:"unit_price"`
	Subtotal    float64 `json:"subtotal"`
}

type EDIIngestRequest struct {
	RawPayload      string `json:"raw_payload" binding:"required"`
	Format          string `json:"format"` // ANSI_X12 or EDIFACT (auto-detected if empty)
	AutoCreateOrder bool   `json:"auto_create_order"`
}

type EDIIngestResponse struct {
	Format             string        `json:"format"`
	DocumentType       string        `json:"document_type"`
	TransactionCode    string        `json:"transaction_code"`
	SenderID           string        `json:"sender_id"`
	ReceiverID         string        `json:"receiver_id"`
	ControlNumber      string        `json:"control_number"`
	PONumber           string        `json:"po_number"`
	OrderDate          string        `json:"order_date"`
	Currency           string        `json:"currency"`
	TotalAmount        float64       `json:"total_amount"`
	LineItemsCount     int           `json:"line_items_count"`
	LineItems          []EDILineItem `json:"line_items"`
	CreatedRecordID    *uint         `json:"created_record_id,omitempty"`
	RecordType         string        `json:"record_type,omitempty"`
	Status             string        `json:"status"` // SUCCESS, WARNING, ERROR
	Message            string        `json:"message"`
	SegmentsProcessed  int           `json:"segments_processed"`
	ValidationWarnings []string      `json:"validation_warnings,omitempty"`
	ParsedAt           string        `json:"parsed_at"`
}

type EDIGenerateRequest struct {
	DocumentType   string        `json:"document_type" binding:"required"` // 810 (Invoice), 850 (PO), 855 (PO Ack), 856 (ASN)
	Format         string        `json:"format"`                          // ANSI_X12 or EDIFACT (default ANSI_X12)
	SenderID       string        `json:"sender_id"`
	ReceiverID     string        `json:"receiver_id"`
	DocumentNumber string        `json:"document_number" binding:"required"`
	Date           string        `json:"date"`
	Currency       string        `json:"currency"`
	CarrierCode    string        `json:"carrier_code,omitempty"`
	TrackingNumber string        `json:"tracking_number,omitempty"`
	LineItems      []EDILineItem `json:"line_items"`
	TotalAmount    float64       `json:"total_amount"`
}

type EDIGenerateResponse struct {
	DocumentType   string `json:"document_type"`
	Format         string `json:"format"`
	ControlNumber  string `json:"control_number"`
	RawPayload     string `json:"raw_payload"`
	SegmentCount   int    `json:"segment_count"`
	DocumentNumber string `json:"document_number"`
	GeneratedAt    string `json:"generated_at"`
}

// IngestEDI parses raw EDI stream (ANSI X12 or EDIFACT)
func IngestEDI(raw string, format string, autoCreate bool, db *gorm.DB, companyID uint) *EDIIngestResponse {
	raw = strings.TrimSpace(raw)
	detectedFormat := format
	if detectedFormat == "" {
		if strings.HasPrefix(raw, "ISA") || strings.Contains(raw, "*") && strings.Contains(raw, "~") {
			detectedFormat = "ANSI_X12"
		} else if strings.HasPrefix(raw, "UNA") || strings.HasPrefix(raw, "UNB") || strings.Contains(raw, "+") && strings.Contains(raw, "'") {
			detectedFormat = "EDIFACT"
		} else {
			detectedFormat = "ANSI_X12"
		}
	}

	res := &EDIIngestResponse{
		Format:    detectedFormat,
		Currency:  "IDR",
		ParsedAt:  time.Now().Format(time.RFC3339),
		Status:    "SUCCESS",
		LineItems: make([]EDILineItem, 0),
	}

	if detectedFormat == "ANSI_X12" {
		parseANSIX12(raw, res)
	} else {
		parseEDIFACT(raw, res)
	}

	if autoCreate && res.Status == "SUCCESS" && db != nil && len(res.LineItems) > 0 {
		// Stage a draft SaleOrder in database
		soName := "EDI-" + res.PONumber
		if res.PONumber == "" {
			soName = fmt.Sprintf("EDI-%d", time.Now().Unix())
		}
		newOrder := sale.SaleOrder{
			Name:          soName,
			PartnerID:     1, // Default EDI partner/generic customer
			DateOrder:     res.OrderDate,
			State:         "draft",
			AmountUntaxed: res.TotalAmount,
			AmountTax:     res.TotalAmount * 0.11,
			AmountTotal:   res.TotalAmount * 1.11,
			Note:          fmt.Sprintf("Auto-generated via Universal EDI Ingest (%s - %s) Ref: %s", res.Format, res.DocumentType, res.ControlNumber),
		}
		newOrder.CompanyID = companyID

		if err := db.Create(&newOrder).Error; err == nil {
			res.CreatedRecordID = &newOrder.ID
			res.RecordType = "sale.order"
			res.Message += fmt.Sprintf(" Automatically staged Draft Sales Order #%d (%s).", newOrder.ID, newOrder.Name)
		}
	}

	return res
}

func parseANSIX12(raw string, res *EDIIngestResponse) {
	// Standard ANSI X12 segments are separated by ~ or newline
	cleanRaw := strings.ReplaceAll(raw, "\r\n", "")
	cleanRaw = strings.ReplaceAll(cleanRaw, "\n", "")
	segments := strings.Split(cleanRaw, "~")

	res.SegmentsProcessed = len(segments)
	currentLine := EDILineItem{}

	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		elems := strings.Split(seg, "*")
		tag := elems[0]

		switch tag {
		case "ISA":
			if len(elems) >= 14 {
				res.SenderID = strings.TrimSpace(elems[6])
				res.ReceiverID = strings.TrimSpace(elems[8])
				res.ControlNumber = strings.TrimSpace(elems[13])
			}
		case "ST":
			if len(elems) >= 2 {
				res.TransactionCode = elems[1]
				switch elems[1] {
				case "850":
					res.DocumentType = "PURCHASE_ORDER_850"
				case "810":
					res.DocumentType = "INVOICE_810"
				case "855":
					res.DocumentType = "PO_ACK_855"
				case "856":
					res.DocumentType = "SHIP_NOTICE_ASN_856"
				default:
					res.DocumentType = "EDI_TRANSACTION_" + elems[1]
				}
			}
		case "BEG": // Purchase order header
			if len(elems) >= 4 {
				res.PONumber = elems[3]
			}
			if len(elems) >= 6 && elems[5] != "" {
				res.OrderDate = elems[5]
			} else {
				res.OrderDate = time.Now().Format("2006-01-02")
			}
		case "BIG": // Commercial Invoice header
			if len(elems) >= 3 {
				res.OrderDate = elems[1]
				res.PONumber = elems[2] // Invoice # or PO #
			}
		case "CUR":
			if len(elems) >= 2 {
				res.Currency = elems[1]
			}
		case "PO1", "IT1":
			if currentLine.LineNumber > 0 {
				res.LineItems = append(res.LineItems, currentLine)
			}
			lineNum, _ := strconv.Atoi(elems[1])
			qty, _ := strconv.ParseFloat(elems[2], 64)
			unit := "PCS"
			if len(elems) >= 4 {
				unit = elems[3]
			}
			price := 0.0
			if len(elems) >= 5 {
				price, _ = strconv.ParseFloat(elems[4], 64)
			}
			pCode := "PROD-GENERIC"
			if len(elems) >= 8 {
				pCode = elems[7]
			}

			currentLine = EDILineItem{
				LineNumber:  lineNum,
				ProductCode: pCode,
				Description: "EDI Item " + pCode,
				Quantity:    qty,
				Unit:        unit,
				UnitPrice:   price,
				Subtotal:    qty * price,
			}
		case "PID":
			if len(elems) >= 6 && elems[5] != "" {
				currentLine.Description = elems[5]
			}
		case "TDS": // Total Monetary Value
			if len(elems) >= 2 {
				tot, _ := strconv.ParseFloat(elems[1], 64)
				// ANSI X12 TDS is frequently in cents (or nominal)
				if tot > 0 && res.TotalAmount == 0 {
					res.TotalAmount = tot
				}
			}
		}
	}

	if currentLine.LineNumber > 0 {
		res.LineItems = append(res.LineItems, currentLine)
	}

	res.LineItemsCount = len(res.LineItems)
	if res.TotalAmount == 0 {
		calcTotal := 0.0
		for _, item := range res.LineItems {
			calcTotal += item.Subtotal
		}
		res.TotalAmount = calcTotal
	}

	if res.DocumentType == "" {
		res.DocumentType = "PURCHASE_ORDER_850"
	}
	res.Message = fmt.Sprintf("Successfully ingested ANSI X12 %s containing %d line items. Total: %s %.2f.",
		res.DocumentType, res.LineItemsCount, res.Currency, res.TotalAmount)
}

func parseEDIFACT(raw string, res *EDIIngestResponse) {
	cleanRaw := strings.ReplaceAll(raw, "\r\n", "")
	cleanRaw = strings.ReplaceAll(cleanRaw, "\n", "")
	segments := strings.Split(cleanRaw, "'")

	res.SegmentsProcessed = len(segments)
	currentLine := EDILineItem{}
	lineSeq := 1

	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		elems := strings.Split(seg, "+")
		tag := elems[0]

		switch tag {
		case "UNB":
			if len(elems) >= 6 {
				res.SenderID = elems[2]
				res.ReceiverID = elems[3]
				res.ControlNumber = elems[5]
			}
		case "BGM":
			if len(elems) >= 3 {
				code := elems[1]
				res.PONumber = elems[2]
				switch code {
				case "220", "ORDERS":
					res.DocumentType = "PURCHASE_ORDER_ORDERS"
					res.TransactionCode = "ORDERS"
				case "380", "INVOIC":
					res.DocumentType = "INVOICE_INVOIC"
					res.TransactionCode = "INVOIC"
				case "351", "DESADV":
					res.DocumentType = "SHIP_NOTICE_DESADV"
					res.TransactionCode = "DESADV"
				default:
					res.DocumentType = "EDIFACT_" + code
				}
			}
		case "DTM":
			if len(elems) >= 2 {
				dtmParts := strings.Split(elems[1], ":")
				if len(dtmParts) >= 2 {
					res.OrderDate = dtmParts[1]
				}
			}
		case "LIN":
			if currentLine.LineNumber > 0 {
				res.LineItems = append(res.LineItems, currentLine)
			}
			pCode := "PROD-GENERIC"
			if len(elems) >= 4 {
				pParts := strings.Split(elems[3], ":")
				if len(pParts) >= 1 && pParts[0] != "" {
					pCode = pParts[0]
				}
			}
			currentLine = EDILineItem{
				LineNumber:  lineSeq,
				ProductCode: pCode,
				Description: "EDIFACT Item " + pCode,
				Unit:        "PCE",
			}
			lineSeq++
		case "QTY":
			if len(elems) >= 2 {
				qParts := strings.Split(elems[1], ":")
				if len(qParts) >= 2 {
					qty, _ := strconv.ParseFloat(qParts[1], 64)
					currentLine.Quantity = qty
				}
				if len(qParts) >= 3 && qParts[2] != "" {
					currentLine.Unit = qParts[2]
				}
			}
		case "PRI":
			if len(elems) >= 2 {
				pParts := strings.Split(elems[1], ":")
				if len(pParts) >= 2 {
					price, _ := strconv.ParseFloat(pParts[1], 64)
					currentLine.UnitPrice = price
					currentLine.Subtotal = currentLine.Quantity * price
				}
			}
		case "IMD":
			if len(elems) >= 4 {
				currentLine.Description = elems[3]
			}
		}
	}

	if currentLine.LineNumber > 0 {
		res.LineItems = append(res.LineItems, currentLine)
	}

	res.LineItemsCount = len(res.LineItems)
	calcTotal := 0.0
	for _, item := range res.LineItems {
		calcTotal += item.Subtotal
	}
	res.TotalAmount = calcTotal

	if res.DocumentType == "" {
		res.DocumentType = "PURCHASE_ORDER_ORDERS"
	}
	res.Message = fmt.Sprintf("Successfully ingested UN/EDIFACT %s containing %d line items. Total: %s %.2f.",
		res.DocumentType, res.LineItemsCount, res.Currency, res.TotalAmount)
}

// GenerateEDI creates an ANSI X12 or EDIFACT compliant stream
func GenerateEDI(req *EDIGenerateRequest) *EDIGenerateResponse {
	format := req.Format
	if format == "" {
		format = "ANSI_X12"
	}
	sender := req.SenderID
	if sender == "" {
		sender = "GOERP_GLOBAL"
	}
	receiver := req.ReceiverID
	if receiver == "" {
		receiver = "PARTNER_EDI"
	}
	ctrl := fmt.Sprintf("%09d", time.Now().Unix()%1000000000)
	dateStamp := time.Now().Format("20060102")
	timeStamp := time.Now().Format("1504")

	var sb strings.Builder
	segCount := 0

	if format == "ANSI_X12" {
		// Header Envelope
		sb.WriteString(fmt.Sprintf("ISA*00*          *00*          *ZZ*%-15s*ZZ*%-15s*%s*%s*U*00401*%s*0*P*>~\n",
			sender, receiver, time.Now().Format("060102"), timeStamp, ctrl))
		sb.WriteString(fmt.Sprintf("GS*%s*%s*%s*%s*%s*%s*X*004010~\n",
			getGSTag(req.DocumentType), sender, receiver, dateStamp, timeStamp, ctrl))

		stNum := "0001"
		switch req.DocumentType {
		case "810", "INVOICE":
			sb.WriteString(fmt.Sprintf("ST*810*%s~\n", stNum))
			sb.WriteString(fmt.Sprintf("BIG*%s*%s*%s~\n", dateStamp, req.DocumentNumber, req.DocumentNumber))
			sb.WriteString(fmt.Sprintf("CUR*%s~\n", defaultVal(req.Currency, "IDR")))
			sb.WriteString(fmt.Sprintf("N1*RE*%s~\n", sender))
			sb.WriteString(fmt.Sprintf("N1*BT*%s~\n", receiver))
			segCount += 5

			for i, it := range req.LineItems {
				idx := i + 1
				if it.LineNumber > 0 {
					idx = it.LineNumber
				}
				sb.WriteString(fmt.Sprintf("IT1*%d*%.0f*%s*%.2f**VN*%s~\n",
					idx, it.Quantity, defaultVal(it.Unit, "EA"), it.UnitPrice, it.ProductCode))
				if it.Description != "" {
					sb.WriteString(fmt.Sprintf("PID*F****%s~\n", it.Description))
					segCount++
				}
				segCount++
			}
			sb.WriteString(fmt.Sprintf("TDS*%.0f~\n", req.TotalAmount))
			sb.WriteString(fmt.Sprintf("CTT*%d~\n", len(req.LineItems)))
			segCount += 2

		case "856", "ASN", "DESADV":
			sb.WriteString(fmt.Sprintf("ST*856*%s~\n", stNum))
			sb.WriteString(fmt.Sprintf("BSN*00*%s*%s*%s~\n", req.DocumentNumber, dateStamp, timeStamp))
			sb.WriteString(fmt.Sprintf("TD5*B*2*%s*M~\n", defaultVal(req.CarrierCode, "JNE_EXPRESS")))
			sb.WriteString(fmt.Sprintf("REF*BM*%s~\n", defaultVal(req.TrackingNumber, "TRK-EDI-99812")))
			segCount += 4

			for i, it := range req.LineItems {
				idx := i + 1
				sb.WriteString(fmt.Sprintf("HL*%d*1*I~\n", idx))
				sb.WriteString(fmt.Sprintf("LIN*%d*BP*%s~\n", idx, it.ProductCode))
				sb.WriteString(fmt.Sprintf("SN1**%.0f*%s~\n", it.Quantity, defaultVal(it.Unit, "EA")))
				segCount += 3
			}
			sb.WriteString(fmt.Sprintf("CTT*%d~\n", len(req.LineItems)))
			segCount++

		case "855", "PO_ACK":
			sb.WriteString(fmt.Sprintf("ST*855*%s~\n", stNum))
			sb.WriteString(fmt.Sprintf("BAK*00*AC*%s*%s~\n", req.DocumentNumber, dateStamp))
			segCount += 2
			for i, it := range req.LineItems {
				idx := i + 1
				sb.WriteString(fmt.Sprintf("PO1*%d*%.0f*%s*%.2f**VN*%s~\n",
					idx, it.Quantity, defaultVal(it.Unit, "EA"), it.UnitPrice, it.ProductCode))
				sb.WriteString("ACK*IA~\n")
				segCount += 2
			}
			sb.WriteString(fmt.Sprintf("CTT*%d~\n", len(req.LineItems)))
			segCount++

		default: // 850 Purchase Order default
			sb.WriteString(fmt.Sprintf("ST*850*%s~\n", stNum))
			sb.WriteString(fmt.Sprintf("BEG*00*SA*%s**%s~\n", req.DocumentNumber, dateStamp))
			sb.WriteString(fmt.Sprintf("CUR*%s~\n", defaultVal(req.Currency, "IDR")))
			segCount += 3
			for i, it := range req.LineItems {
				idx := i + 1
				sb.WriteString(fmt.Sprintf("PO1*%d*%.0f*%s*%.2f**VN*%s~\n",
					idx, it.Quantity, defaultVal(it.Unit, "EA"), it.UnitPrice, it.ProductCode))
				segCount++
			}
			sb.WriteString(fmt.Sprintf("CTT*%d~\n", len(req.LineItems)))
			segCount++
		}

		sb.WriteString(fmt.Sprintf("SE*%d*%s~\n", segCount+2, stNum))
		sb.WriteString(fmt.Sprintf("GE*1*%s~\n", ctrl))
		sb.WriteString(fmt.Sprintf("IEA*1*%s~", ctrl))
		segCount += 3

	} else {
		// UN/EDIFACT Format
		sb.WriteString("UNA:+.? '\n")
		sb.WriteString(fmt.Sprintf("UNB+UNOC:3+%s:ZZZ+%s:ZZZ+%s:%s+%s'\n", sender, receiver, dateStamp, timeStamp, ctrl))

		switch req.DocumentType {
		case "810", "INVOICE", "INVOIC":
			sb.WriteString(fmt.Sprintf("BGM+380+%s+9'\n", req.DocumentNumber))
			sb.WriteString(fmt.Sprintf("DTM+137:%s:102'\n", dateStamp))
			sb.WriteString(fmt.Sprintf("NAD+SU+%s'\n", sender))
			sb.WriteString(fmt.Sprintf("NAD+BY+%s'\n", receiver))
			for i, it := range req.LineItems {
				idx := i + 1
				sb.WriteString(fmt.Sprintf("LIN+%d++%s:IN'\n", idx, it.ProductCode))
				sb.WriteString(fmt.Sprintf("QTY+47:%.0f:%s'\n", it.Quantity, defaultVal(it.Unit, "PCE")))
				sb.WriteString(fmt.Sprintf("MOA+203:%.2f'\n", it.Subtotal))
				sb.WriteString(fmt.Sprintf("PRI+AAA:%.2f'\n", it.UnitPrice))
			}
			sb.WriteString(fmt.Sprintf("UNT*%d*%s'\n", len(req.LineItems)*4+5, ctrl))

		default:
			sb.WriteString(fmt.Sprintf("BGM+220+%s+9'\n", req.DocumentNumber))
			sb.WriteString(fmt.Sprintf("DTM+137:%s:102'\n", dateStamp))
			for i, it := range req.LineItems {
				idx := i + 1
				sb.WriteString(fmt.Sprintf("LIN+%d++%s:IN'\n", idx, it.ProductCode))
				sb.WriteString(fmt.Sprintf("QTY+21:%.0f:%s'\n", it.Quantity, defaultVal(it.Unit, "PCE")))
			}
			sb.WriteString(fmt.Sprintf("UNT*%d*%s'\n", len(req.LineItems)*2+3, ctrl))
		}
		sb.WriteString(fmt.Sprintf("UNZ+1+%s'", ctrl))
		segCount = strings.Count(sb.String(), "'")
	}

	return &EDIGenerateResponse{
		DocumentType:   req.DocumentType,
		Format:         format,
		ControlNumber:  ctrl,
		RawPayload:     sb.String(),
		SegmentCount:   segCount,
		DocumentNumber: req.DocumentNumber,
		GeneratedAt:    time.Now().Format(time.RFC3339),
	}
}

func getGSTag(docType string) string {
	switch docType {
	case "810", "INVOICE":
		return "IN"
	case "856", "ASN":
		return "SH"
	case "855", "PO_ACK":
		return "PR"
	default:
		return "PO"
	}
}

func defaultVal(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// RegisterEDIRoutes mounts EDI Ingestion and Generation routes
func RegisterEDIRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	ediGroup := router.Group("/edi")
	ediGroup.Use(authMiddleware)
	{
		ediGroup.POST("/ingest", func(c *gin.Context) {
			_, companyID, _ := middleware.GetUserContext(c)
			var req EDIIngestRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := IngestEDI(req.RawPayload, req.Format, req.AutoCreateOrder, db, companyID)
			c.JSON(http.StatusOK, res)
		})

		ediGroup.POST("/generate", func(c *gin.Context) {
			var req EDIGenerateRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GenerateEDI(&req)
			c.JSON(http.StatusOK, res)
		})

		ediGroup.GET("/supported", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"standards": []string{"ANSI_X12", "UN_EDIFACT"},
				"supported_transaction_sets": []gin.H{
					{"code": "850", "name": "Purchase Order", "direction": "Inbound/Outbound", "edifact_equiv": "ORDERS"},
					{"code": "855", "name": "Purchase Order Acknowledgment", "direction": "Outbound", "edifact_equiv": "ORDRSP"},
					{"code": "856", "name": "Advance Ship Notice (ASN)", "direction": "Outbound", "edifact_equiv": "DESADV"},
					{"code": "810", "name": "Commercial Invoice", "direction": "Inbound/Outbound", "edifact_equiv": "INVOIC"},
				},
				"version": "v1.46.0 Universal Gateway",
			})
		})
	}
}
