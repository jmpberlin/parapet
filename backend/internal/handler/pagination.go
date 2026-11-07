package handler

import (
	"errors"
	"net/http"
	"strconv"
)

func getPaginationParams(r *http.Request, defaultLimit int) (page, limit int, err error) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	if pageStr == "" {
		pageStr = "1"
	}
	if limitStr == "" {
		limitStr = strconv.Itoa(defaultLimit)
	}

	page, err = strconv.Atoi(pageStr)
	if err != nil || page < 1 || page > 1000000 {
		return 0, 0, errors.New("invalid page parameter")
	}

	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		return 0, 0, errors.New("invalid limit parameter")
	}

	return page, limit, nil
}

func calculateTotalPages(total, limit int) int {
	if total == 0 {
		return 1
	}
	return (total + limit - 1) / limit
}
