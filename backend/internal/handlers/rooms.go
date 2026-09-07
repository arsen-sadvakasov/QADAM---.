package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// RoomsHandler содержит HTTP-хендлеры для /api/v1/rooms/*
// (раздел 35 API Plan). Чтение — все авторизованные, запись — Admin.
type RoomsHandler struct {
	rooms repositories.RoomRepository
}

// NewRoomsHandler создаёт RoomsHandler с внедрённым RoomRepository.
func NewRoomsHandler(rooms repositories.RoomRepository) *RoomsHandler {
	return &RoomsHandler{rooms: rooms}
}

type roomDTO struct {
	ID       string  `json:"id"`
	Number   string  `json:"number"`
	Name     *string `json:"name"`
	Type     string  `json:"type"`
	Floor    *int    `json:"floor"`
	Building *string `json:"building"`
	Notes    *string `json:"notes"`
}

func roomDTOFromModel(rm *models.Room) roomDTO {
	return roomDTO{
		ID:       rm.ID,
		Number:   rm.Number,
		Name:     rm.Name,
		Type:     string(rm.Type),
		Floor:    rm.Floor,
		Building: rm.Building,
		Notes:    rm.Notes,
	}
}

// List обрабатывает GET /api/v1/rooms. Все авторизованные.
func (h *RoomsHandler) List(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.rooms.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	dtos := make([]roomDTO, 0, len(rooms))
	for _, rm := range rooms {
		dtos = append(dtos, roomDTOFromModel(rm))
	}
	writeJSON(w, http.StatusOK, map[string]any{"rooms": dtos})
}

// Get обрабатывает GET /api/v1/rooms/{id}.
func (h *RoomsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rm, err := h.rooms.FindByID(r.Context(), id)
	if err != nil {
		handleNotFoundOr500(w, err, "room not found")
		return
	}
	writeJSON(w, http.StatusOK, roomDTOFromModel(rm))
}

type roomWriteRequest struct {
	Number   string  `json:"number"`
	Name     *string `json:"name"`
	Type     string  `json:"type"`
	Floor    *int    `json:"floor"`
	Building *string `json:"building"`
	Notes    *string `json:"notes"`
}

var validRoomTypes = map[string]bool{"lecture": true, "lab": true, "computer": true, "other": true}

// Create обрабатывает POST /api/v1/rooms. Admin only.
func (h *RoomsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req roomWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Number == "" {
		writeError(w, http.StatusBadRequest, "number is required")
		return
	}
	roomType := req.Type
	if roomType == "" {
		roomType = "other"
	}
	if !validRoomTypes[roomType] {
		writeError(w, http.StatusBadRequest, "invalid room type")
		return
	}

	rm := &models.Room{
		Number:   req.Number,
		Name:     req.Name,
		Type:     models.RoomType(roomType),
		Floor:    req.Floor,
		Building: req.Building,
		Notes:    req.Notes,
	}
	id, err := h.rooms.Create(r.Context(), rm)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	rm.ID = id
	writeJSON(w, http.StatusCreated, roomDTOFromModel(rm))
}

// Update обрабатывает PATCH /api/v1/rooms/{id}. Admin only.
func (h *RoomsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := h.rooms.FindByID(r.Context(), id)
	if err != nil {
		handleNotFoundOr500(w, err, "room not found")
		return
	}

	var req roomWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Number != "" {
		existing.Number = req.Number
	}
	if req.Type != "" {
		if !validRoomTypes[req.Type] {
			writeError(w, http.StatusBadRequest, "invalid room type")
			return
		}
		existing.Type = models.RoomType(req.Type)
	}
	existing.Name = req.Name
	existing.Floor = req.Floor
	existing.Building = req.Building
	existing.Notes = req.Notes

	if err := h.rooms.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, roomDTOFromModel(existing))
}
