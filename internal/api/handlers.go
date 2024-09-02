package api

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/Cdaprod/registry-service/internal/registry"
    "github.com/Cdaprod/registry-service/internal/storage"
    "github.com/gorilla/mux"
    "go.uber.org/zap"
)

type Handler struct {
    store  *storage.MemoryStorage
    logger *zap.Logger
}

func NewHandler(store *storage.MemoryStorage, logger *zap.Logger) *Handler {
    return &Handler{
        store:  store,
        logger: logger,
    }
}

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
    var item registry.Item
    if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
        h.logger.Error("Failed to decode request body", zap.Error(err))
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    createdItem, err := h.store.CreateItem(&item)
    if err != nil {
        h.logger.Error("Failed to create item", zap.Error(err))
        http.Error(w, "Failed to create item", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(createdItem)
}

func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
    params := mux.Vars(r)
    id := params["id"]

    item, err := h.store.GetItem(id)
    if err != nil {
        h.logger.Error("Failed to get item", zap.Error(err))
        http.Error(w, "Item not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(item)
}

func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
    params := mux.Vars(r)
    id := params["id"]

    var item registry.Item
    if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
        h.logger.Error("Failed to decode request body", zap.Error(err))
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    item.ID = id
    updatedItem, err := h.store.UpdateItem(&item)
    if err != nil {
        h.logger.Error("Failed to update item", zap.Error(err))
        http.Error(w, "Failed to update item", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(updatedItem)
}

func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
    params := mux.Vars(r)
    id := params["id"]

    if err := h.store.DeleteItem(id); err != nil {
        h.logger.Error("Failed to delete item", zap.Error(err))
        http.Error(w, "Failed to delete item", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

    items, err := h.store.List(limit, offset)
    if err != nil {
        h.logger.Error("Failed to list items", zap.Error(err))
        http.Error(w, "Failed to list items", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(items)
}