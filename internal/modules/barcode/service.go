package barcode

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/stock"
	"erp-golang/internal/modules/stock_inventory"

	"gorm.io/gorm"
)

type Service struct {
	db   *gorm.DB
	repo *Repository
}

func NewService(db *gorm.DB, repo *Repository) *Service {
	return &Service{
		db:   db,
		repo: repo,
	}
}

// ---------------------------------------------------------------------------
// Universal Barcode Scanner Engine
// ---------------------------------------------------------------------------

// Scan resolves any scanned barcode string and applies context-specific actions
func (s *Service) Scan(req BarcodeScanRequest, companyID uint, userID uint) (*BarcodeScanResult, error) {
	code := strings.TrimSpace(req.Barcode)
	if code == "" {
		return nil, errors.New("barcode tidak boleh kosong")
	}

	qty := req.Qty
	if qty <= 0 {
		qty = 1.0
	}

	res := &BarcodeScanResult{
		Found:   false,
		Barcode: code,
		Type:    "unknown",
		Message: "Barcode tidak ditemukan dalam database",
	}

	// 1. Check Product first (by barcode or default_code/SKU)
	prod, err := s.repo.FindProductByBarcode(code, companyID)
	if err == nil && prod != nil {
		res.Found = true
		res.Type = "product"
		res.Product = prod
		res.Message = fmt.Sprintf("Produk terdeteksi: %s (%s)", prod.Name, prod.DefaultCode)

		// Context-specific actions
		switch req.Context {
		case "pos":
			// Formatted for POS Cashier Cart
			res.ActionTaken = "pos_item_prepared"
			res.Data = map[string]interface{}{
				"product_id":     prod.ID,
				"product_name":   prod.Name,
				"barcode":        prod.Barcode,
				"default_code":   prod.DefaultCode,
				"price_unit":     prod.ListPrice,
				"qty":            qty,
				"price_subtotal": prod.ListPrice * qty,
				"uom":            prod.UoM.Name,
			}
			res.Message = fmt.Sprintf("Produk '%s' siap ditambahkan ke keranjang POS kasir (Qty: %.0f, Harga: Rp %.0f)",
				prod.Name, qty, prod.ListPrice)

		case "stock_inventory":
			if req.TargetID > 0 {
				var inv stock_inventory.StockInventory
				if err := s.db.Preload("Lines").Where("id = ?", req.TargetID).First(&inv).Error; err == nil {
					// Check if product line already exists
					var matchedLine *stock_inventory.StockInventoryLine
					for i := range inv.Lines {
						if inv.Lines[i].ProductID == prod.ID {
							matchedLine = &inv.Lines[i]
							break
						}
					}

					if matchedLine != nil {
						// Increment or set counted quantity
						if req.Action == "set" {
							matchedLine.CountedQty = qty
						} else {
							matchedLine.CountedQty += qty
						}
						matchedLine.DifferenceQty = matchedLine.CountedQty - matchedLine.TheoreticalQty
						matchedLine.VarianceValue = matchedLine.DifferenceQty * matchedLine.CostPrice
						s.db.Save(matchedLine)
						res.ActionTaken = "inventory_line_incremented"
						res.Message = fmt.Sprintf("Stok fisik '%s' diperbarui menjadi %.2f (Selisih: %.2f)",
							prod.Name, matchedLine.CountedQty, matchedLine.DifferenceQty)
						res.Data = matchedLine
					} else {
						// Add unexpected physical product line
						newLine := stock_inventory.StockInventoryLine{
							InventoryID:    inv.ID,
							ProductID:      prod.ID,
							TheoreticalQty: 0,
							CountedQty:     qty,
							DifferenceQty:  qty,
							CostPrice:      prod.StandardPrice,
							VarianceValue:  qty * prod.StandardPrice,
							Reason:         "Ditemukan via pemindaian barcode fisik",
							State:          "draft",
						}
						newLine.CompanyID = companyID
						s.db.Create(&newLine)
						res.ActionTaken = "inventory_line_created"
						res.Message = fmt.Sprintf("Item baru '%s' ditambahkan ke stock opname dengan hasil hitung %.2f", prod.Name, qty)
						res.Data = newLine
					}

					// Recalculate inventory totals
					var allLines []stock_inventory.StockInventoryLine
					s.db.Where("inventory_id = ?", inv.ID).Find(&allLines)
					var totTheo, totCount, totDiff, totLoss, totGain float64
					for _, l := range allLines {
						totTheo += l.TheoreticalQty
						totCount += l.CountedQty
						totDiff += l.DifferenceQty
						if l.VarianceValue < 0 {
							totLoss += -l.VarianceValue
						} else {
							totGain += l.VarianceValue
						}
					}
					inv.TotalTheoreticalQty = totTheo
					inv.TotalCountedQty = totCount
					inv.TotalDifferenceQty = totDiff
					inv.TotalLossValue = totLoss
					inv.TotalGainValue = totGain
					s.db.Save(&inv)
				}
			}

		default:
			res.ActionTaken = "product_lookup"
		}
		return res, nil
	}

	// 2. Check Lot / Serial Number
	lot, err := s.repo.FindLotByBarcode(code, companyID)
	if err == nil && lot != nil {
		res.Found = true
		res.Type = "lot"
		res.Lot = lot
		res.Message = fmt.Sprintf("Nomor Lot/Serial terdeteksi: %s (Produk: %s, Qty: %.2f)",
			lot.Name, lot.Product.Name, lot.Qty)
		res.ActionTaken = "lot_lookup"
		return res, nil
	}

	// 3. Check Stock Location (Shelf / Bin)
	loc, err := s.repo.FindLocationByBarcode(code, companyID)
	if err == nil && loc != nil {
		res.Found = true
		res.Type = "location"
		res.Location = loc
		res.Message = fmt.Sprintf("Lokasi gudang terdeteksi: %s (%s)", loc.CompleteName, loc.Usage)
		res.ActionTaken = "location_lookup"
		return res, nil
	}

	// 4. Check Stock Picking (Transfer/Receipt/Delivery Order)
	picking, err := s.repo.FindPickingByBarcode(code, companyID)
	if err == nil && picking != nil {
		res.Found = true
		res.Type = "picking"
		res.Picking = picking
		res.Message = fmt.Sprintf("Dokumen transfer/picking terdeteksi: %s (%s)", picking.Name, picking.State)
		res.ActionTaken = "picking_lookup"
		return res, nil
	}

	return res, nil
}

// ---------------------------------------------------------------------------
// Printable Barcode & Label Generation
// ---------------------------------------------------------------------------

// GenerateLabelSVG generates professional printable vector SVG label with Code128
func (s *Service) GenerateLabelSVG(entityType string, id uint, companyID uint) (string, error) {
	switch entityType {
	case "product":
		var prod product.Product
		q := s.db.Preload("Category").Preload("UoM").Where("id = ?", id)
		if companyID > 0 {
			q = q.Where("company_id = ?", companyID)
		}
		if err := q.First(&prod).Error; err != nil {
			return "", errors.New("produk tidak ditemukan")
		}

		barcodeStr := prod.Barcode
		if barcodeStr == "" {
			barcodeStr = prod.DefaultCode
		}
		if barcodeStr == "" {
			barcodeStr = fmt.Sprintf("PRD%06d", prod.ID)
		}

		priceStr := fmt.Sprintf("Rp %.0f", prod.ListPrice)
		subStr := prod.DefaultCode
		if prod.Category.Name != "" {
			subStr += " (" + prod.Category.Name + ")"
		}

		return GenerateCode128SVG(barcodeStr, prod.Name, subStr, priceStr), nil

	case "lot":
		lot, err := s.repo.GetLotByID(id, companyID)
		if err != nil {
			return "", errors.New("data lot/serial number tidak ditemukan")
		}

		code := lot.Barcode
		if code == "" {
			code = lot.Name
		}
		prodName := "Item Material"
		if lot.Product != nil {
			prodName = lot.Product.Name
		}

		subStr := fmt.Sprintf("LOT: %s", lot.Name)
		expStr := ""
		if lot.ExpirationDate != "" {
			expStr = fmt.Sprintf("EXP: %s", lot.ExpirationDate)
		}

		return GenerateCode128SVG(code, prodName, subStr, expStr), nil

	case "location":
		var loc stock.StockLocation
		q := s.db.Preload("Warehouse").Where("id = ?", id)
		if companyID > 0 {
			q = q.Where("company_id = ?", companyID)
		}
		if err := q.First(&loc).Error; err != nil {
			return "", errors.New("lokasi rak gudang tidak ditemukan")
		}

		code := fmt.Sprintf("LOC-%s", strings.ReplaceAll(loc.CompleteName, "/", "-"))
		whName := "GUDANG UTAMA"
		if loc.Warehouse != nil {
			whName = loc.Warehouse.Name
		}

		return GenerateCode128SVG(code, loc.CompleteName, whName, strings.ToUpper(loc.Usage)), nil

	default:
		return "", fmt.Errorf("tipe label '%s' tidak didukung (gunakan: product, lot, location)", entityType)
	}
}

// ---------------------------------------------------------------------------
// Lot & Serial Number Operations
// ---------------------------------------------------------------------------

func (s *Service) ListLots(productID uint, companyID uint, search string) ([]StockProductionLot, error) {
	return s.repo.ListLots(productID, companyID, search)
}

func (s *Service) CreateLot(req CreateLotRequest, companyID uint) (*StockProductionLot, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		year := time.Now().Format("2006")
		var count int64
		s.db.Model(&StockProductionLot{}).Where("company_id = ?", companyID).Count(&count)
		name = fmt.Sprintf("LOT/%s/%05d", year, count+1)
	}

	barcode := strings.TrimSpace(req.Barcode)
	if barcode == "" {
		barcode = name
	}

	lot := &StockProductionLot{
		Name:           name,
		ProductID:      req.ProductID,
		Barcode:        barcode,
		Ref:            req.Ref,
		ExpirationDate: req.ExpirationDate,
		AlertDate:      req.AlertDate,
		RemovalDate:    req.RemovalDate,
		Qty:            req.Qty,
		LocationID:     req.LocationID,
		Notes:          req.Notes,
	}
	lot.CompanyID = companyID

	if err := s.repo.CreateLot(lot); err != nil {
		return nil, err
	}

	return s.repo.GetLotByID(lot.ID, companyID)
}

func (s *Service) GetLot(id uint, companyID uint) (*StockProductionLot, error) {
	return s.repo.GetLotByID(id, companyID)
}

func (s *Service) GetStats(companyID uint) (*BarcodeStats, error) {
	return s.repo.GetStats(companyID)
}
