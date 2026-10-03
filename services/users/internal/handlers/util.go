package handlers

import "github.com/google/uuid"

func mustUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}
