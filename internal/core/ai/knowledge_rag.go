package ai

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type KnowledgeArticle struct {
	ArticleID string   `json:"article_id"`
	Title     string   `json:"title"`
	Category  string   `json:"category"` // "warranty", "sop", "product_specs", "returns", "invoice_tax"
	Content   string   `json:"content"`
	Tags      []string `json:"tags"`
}

type KnowledgeChunk struct {
	ArticleID      string  `json:"article_id"`
	Title          string  `json:"title"`
	Category       string  `json:"category"`
	Snippet        string  `json:"snippet"`
	RelevanceScore float64 `json:"relevance_score"`
}

type KnowledgeQueryRequest struct {
	Query    string `json:"query" binding:"required"`
	TopK     int    `json:"top_k"`
	Category string `json:"category"`
}

type KnowledgeQueryResponse struct {
	Query             string           `json:"query"`
	Chunks            []KnowledgeChunk `json:"chunks"`
	SynthesizedAnswer string           `json:"synthesized_answer"`
	SourcesCount      int              `json:"sources_count"`
	ExecutedAt        string           `json:"executed_at"`
}

type CopilotDraftRequest struct {
	TicketID         string `json:"ticket_id"`
	CustomerName     string `json:"customer_name"`
	IssueTitle       string `json:"issue_title" binding:"required"`
	IssueDescription string `json:"issue_description" binding:"required"`
}

type CopilotDraftResponse struct {
	TicketID           string   `json:"ticket_id"`
	DraftSubject       string   `json:"draft_subject"`
	DraftReplyText     string   `json:"draft_reply_text"`
	ReferencedArticles []string `json:"referenced_articles"`
	RecommendedAction  string   `json:"recommended_action"`
	ConfidenceScore    float64  `json:"confidence_score"`
	GeneratedAt        string   `json:"generated_at"`
}

var enterpriseKnowledgeBase = []KnowledgeArticle{
	{
		ArticleID: "KB-WRT-001",
		Title:     "Kebijakan Garansi Hardware & Penggantian Unit RMA",
		Category:  "warranty",
		Content:   "Seluruh produk enterprise (Monitor Gaming, Server Rack, Mesin POS) dilindungi garansi resmi selama 24 bulan sejak tanggal faktur penjualan. Kerusakan akibat cacat pabrik atau fluktuasi komponen internal berhak atas penggantian unit baru (RMA) dalam 7 hari kerja. Pelanggan wajib menyertakan nomor Serial Number dan copy faktur INV/2026.",
		Tags:      []string{"garansi", "rma", "rusak", "penggantian", "hardware", "monitor", "server"},
	},
	{
		ArticleID: "KB-RET-002",
		Title:     "SOP Retur Barang & Penerbitan Nota Kredit Pembatalan Invoice",
		Category:  "returns",
		Content:   "Prosedur pengembalian barang retur penjualan harus didahului dengan pengajuan formulir Retur ke tim Helpdesk. Setelah barang diterima di Gudang Utama melalui dokumen WH/IN/RETUR, Departemen Akuntansi akan menerbitkan Nota Retur Pajak dan memposting Credit Note (Refund) yang otomatis memotong saldo piutang pelanggan.",
		Tags:      []string{"retur", "kembali", "nota kredit", "credit note", "refund", "batal"},
	},
	{
		ArticleID: "KB-MCH-003",
		Title:     "Panduan Pemeliharaan Mesin SMT & Batas Getaran ISO 10816",
		Category:  "product_specs",
		Content:   "Mesin SMT Otomatis wajib dioperasikan pada suhu lingkungan 20-25 derajat Celcius. Batas vibrasi mekanis normal adalah di bawah 4.5 mm/s RMS. Apabila getaran melebihi 7.1 mm/s, sistem IoT akan memicu status CRITICAL dan mesin harus segera dihentikan untuk pelumasan bearing dan kalibrasi spindle.",
		Tags:      []string{"smt", "mesin", "getaran", "vibrasi", "suhu", "spindle", "maintenance"},
	},
	{
		ArticleID: "KB-TAX-004",
		Title:     "Kepatuhan Faktur Pajak Elektronik e-Faktur DJP & PPN 11%",
		Category:  "invoice_tax",
		Content:   "Setiap transaksi penyerahan Barang Kena Pajak (BKP) dikenakan PPN 11% sesuai UU HPP. Nomor Seri Faktur Pajak (NSFP) digenerate secara otomatis melalui integrasi Modul e-Faktur GoERP dengan format 010.002-26.XXXXXXXX dan dilaporkan via SPT Masa PPN sebelum akhir bulan berikutnya.",
		Tags:      []string{"faktur", "pajak", "efaktur", "ppn", "djp", "nsfp", "spt"},
	},
}

// QueryKnowledgeRAG retrieves relevant semantic knowledge and synthesizes an answer
func (c *Client) QueryKnowledgeRAG(req *KnowledgeQueryRequest) *KnowledgeQueryResponse {
	topK := req.TopK
	if topK <= 0 {
		topK = 3
	}

	queryLower := strings.ToLower(req.Query)
	queryTokens := strings.Fields(queryLower)

	type scoredArticle struct {
		article KnowledgeArticle
		score   float64
	}

	var scored []scoredArticle
	for _, art := range enterpriseKnowledgeBase {
		if req.Category != "" && art.Category != req.Category {
			continue
		}

		score := 0.0
		titleLower := strings.ToLower(art.Title)
		contentLower := strings.ToLower(art.Content)

		for _, tok := range queryTokens {
			if len(tok) < 3 {
				continue
			}
			if strings.Contains(titleLower, tok) {
				score += 3.0 // Title match weight
			}
			if strings.Contains(contentLower, tok) {
				score += 1.0 // Content match weight
			}
			for _, tag := range art.Tags {
				if strings.Contains(tag, tok) {
					score += 2.0 // Tag match weight
				}
			}
		}

		if score > 0 {
			scored = append(scored, scoredArticle{article: art, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	var chunks []KnowledgeChunk
	limit := int(math.Min(float64(topK), float64(len(scored))))
	for i := 0; i < limit; i++ {
		art := scored[i].article
		snippet := art.Content
		if len(snippet) > 220 {
			snippet = snippet[:220] + "..."
		}
		chunks = append(chunks, KnowledgeChunk{
			ArticleID:      art.ArticleID,
			Title:          art.Title,
			Category:       art.Category,
			Snippet:        snippet,
			RelevanceScore: math.Round((scored[i].score/10.0)*100) / 100,
		})
	}

	// Fallback if no exact keyword found
	if len(chunks) == 0 {
		art := enterpriseKnowledgeBase[0]
		chunks = append(chunks, KnowledgeChunk{
			ArticleID:      art.ArticleID,
			Title:          art.Title,
			Category:       art.Category,
			Snippet:        art.Content[:180] + "...",
			RelevanceScore: 0.65,
		})
	}

	synth := fmt.Sprintf("Berdasarkan rujukan dokumen resmi (%s): %s",
		chunks[0].Title, chunks[0].Snippet)

	return &KnowledgeQueryResponse{
		Query:             req.Query,
		Chunks:            chunks,
		SynthesizedAnswer: synth,
		SourcesCount:      len(chunks),
		ExecutedAt:        time.Now().Format("2006-01-02 15:04:05"),
	}
}

// DraftSupportCopilotReply automatically drafts a context-aware ticket resolution
func (c *Client) DraftSupportCopilotReply(req *CopilotDraftRequest) *CopilotDraftResponse {
	custName := req.CustomerName
	if custName == "" {
		custName = "Bapak/Ibu Pelanggan"
	}

	// Query RAG for best matching KB article
	ragResp := c.QueryKnowledgeRAG(&KnowledgeQueryRequest{
		Query: req.IssueTitle + " " + req.IssueDescription,
		TopK:  2,
	})

	bestChunk := ragResp.Chunks[0]
	var referenced []string
	for _, ch := range ragResp.Chunks {
		referenced = append(referenced, fmt.Sprintf("%s (%s)", ch.Title, ch.ArticleID))
	}

	replyText := fmt.Sprintf("Halo %s,\n\nTerima kasih telah menghubungi Customer Care GoERP. Terkait kendala yang Anda alami mengenai \"%s\":\n\n%s\n\nLangkah selanjutnya:\n1. Tim teknis kami telah menjadwalkan pengecekan unit sesuai SOP yang berlaku.\n2. Mohon pastikan dokumen pendukung faktur dan serial number telah terlampir.\n\nJika ada pertanyaan lebih lanjut, silakan balas pesan ini.\n\nSalam hangat,\nCustomer Support Operations",
		custName, req.IssueTitle, bestChunk.Snippet)

	subject := fmt.Sprintf("Re: [%s] Penyelesaian Kendala: %s", req.TicketID, req.IssueTitle)

	return &CopilotDraftResponse{
		TicketID:           req.TicketID,
		DraftSubject:       subject,
		DraftReplyText:     replyText,
		ReferencedArticles: referenced,
		RecommendedAction:  "Kirim draf balasan resmi ke pelanggan dan jadwalkan inspeksi teknis RMA.",
		ConfidenceScore:    0.94,
		GeneratedAt:        time.Now().Format("2006-01-02 15:04:05"),
	}
}
