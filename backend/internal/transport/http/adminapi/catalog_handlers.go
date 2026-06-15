package adminapi

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/httperr"
	adminuc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/admin"
)

// maxImageUploadBytes caps the multipart catalog image request (5 MB image +
// form overhead).
const maxImageUploadBytes = 6 * 1024 * 1024

type productDTO struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	ShortDesc   string   `json:"short_desc"`
	LongDesc    string   `json:"long_desc,omitempty"`
	ImageURL    string   `json:"image_url,omitempty"`
	DeeplinkURL string   `json:"deeplink_url,omitempty"`
	Category    string   `json:"category"`
	Problems    []string `json:"problems"`
	Plants      []string `json:"plants"`
	Priority    int32    `json:"priority"`
	Active      bool     `json:"active"`
	PriceRub    *int32   `json:"price_rub,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type productRequest struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	ShortDesc   string   `json:"short_desc"`
	LongDesc    string   `json:"long_desc"`
	ImageURL    string   `json:"image_url"`
	DeeplinkURL string   `json:"deeplink_url"`
	Category    string   `json:"category"`
	Problems    []string `json:"problems"`
	Plants      []string `json:"plants"`
	Priority    int32    `json:"priority"`
	Active      bool     `json:"active"`
	PriceRub    *int32   `json:"price_rub"`
}

func (req productRequest) toInput() adminuc.ProductInput {
	return adminuc.ProductInput{
		Slug:        req.Slug,
		Name:        req.Name,
		ShortDesc:   req.ShortDesc,
		LongDesc:    req.LongDesc,
		ImageURL:    req.ImageURL,
		DeeplinkURL: req.DeeplinkURL,
		Category:    req.Category,
		Problems:    req.Problems,
		Plants:      req.Plants,
		Priority:    req.Priority,
		Active:      req.Active,
		PriceRub:    req.PriceRub,
	}
}

func toProductDTO(v adminuc.ProductView) productDTO {
	return productDTO{
		ID:          v.ID.String(),
		Slug:        v.Slug,
		Name:        v.Name,
		ShortDesc:   v.ShortDesc,
		LongDesc:    v.LongDesc,
		ImageURL:    v.ImageURL,
		DeeplinkURL: v.DeeplinkURL,
		Category:    v.Category,
		Problems:    v.Problems,
		Plants:      v.Plants,
		Priority:    v.Priority,
		Active:      v.Active,
		PriceRub:    v.PriceRub,
		CreatedAt:   v.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   v.UpdatedAt.Format(time.RFC3339),
	}
}

// getProducts — GET /admin/v1/fertilizers.
func (h *Handler) getProducts(w http.ResponseWriter, r *http.Request) {
	views, err := h.svc.ListProducts(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	out := make([]productDTO, 0, len(views))
	for _, v := range views {
		out = append(out, toProductDTO(v))
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"products": out})
}

// getProduct — GET /admin/v1/fertilizers/{id}.
func (h *Handler) getProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httperr.Write(w, r, httperr.ValidationFailed("invalid product id"))
		return
	}
	view, err := h.svc.GetProduct(r.Context(), id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"product": toProductDTO(view)})
}

// postProduct — POST /admin/v1/fertilizers.
func (h *Handler) postProduct(w http.ResponseWriter, r *http.Request) {
	var req productRequest
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	view, err := h.svc.CreateProduct(r.Context(), adminIDFrom(r.Context()), req.toInput(), deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusCreated, map[string]any{"product": toProductDTO(view)})
}

// putProduct — PUT /admin/v1/fertilizers/{id}.
func (h *Handler) putProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httperr.Write(w, r, httperr.ValidationFailed("invalid product id"))
		return
	}
	var req productRequest
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	view, err := h.svc.UpdateProduct(r.Context(), adminIDFrom(r.Context()), id, req.toInput(), deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"product": toProductDTO(view)})
}

// deleteProduct — DELETE /admin/v1/fertilizers/{id}.
func (h *Handler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httperr.Write(w, r, httperr.ValidationFailed("invalid product id"))
		return
	}
	if err := h.svc.DeleteProduct(r.Context(), adminIDFrom(r.Context()), id, deviceMetaFrom(r)); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// getProblemKeys — GET /admin/v1/fertilizers/problems. The closed enum the
// model can produce; the catalog form renders these as choices.
func (h *Handler) getProblemKeys(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusOK, map[string]any{"problems": llm.FertilizerProblemKeys})
}

// postProductImage — POST /admin/v1/fertilizers/image (multipart "file").
func (h *Handler) postProductImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImageUploadBytes)
	if err := r.ParseMultipartForm(maxImageUploadBytes); err != nil { //nolint:gosec // G120: body already bounded by MaxBytesReader above and the form size arg
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			httperr.Write(w, r, httperr.PayloadTooLarge("image is too large (max 5 MB)"))
		} else {
			httperr.Write(w, r, httperr.ValidationFailed("invalid multipart form"))
		}
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httperr.Write(w, r, httperr.ValidationFailed("multipart field 'file' is required"))
		return
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		httperr.Write(w, r, httperr.ValidationFailed("failed to read uploaded file"))
		return
	}
	contentType := header.Header.Get("Content-Type")
	url, err := h.svc.UploadProductImage(r.Context(), adminIDFrom(r.Context()), contentType, data, deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]string{"url": url})
}
