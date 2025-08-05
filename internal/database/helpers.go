package database

import (
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

// StringToPgText converts a string to pgtype.Text
// If the string is empty, it returns an invalid (NULL) pgtype.Text
func StringToPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// StringToPgTextPtr converts a string pointer to pgtype.Text
// If the pointer is nil or points to an empty string, returns invalid pgtype.Text
func StringToPgTextPtr(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// PgTextToString converts pgtype.Text to a regular string
// Returns empty string if the pgtype.Text is invalid (NULL)
func PgTextToString(pt pgtype.Text) string {
	if !pt.Valid {
		return ""
	}
	return pt.String
}

// PgTextToStringPtr converts pgtype.Text to a string pointer
// Returns nil if the pgtype.Text is invalid (NULL)
func PgTextToStringPtr(pt pgtype.Text) *string {
	if !pt.Valid {
		return nil
	}
	return &pt.String
}

// Additional helper functions for other common pgtype conversions

// Int32ToPgInt4 converts int32 to pgtype.Int4
func Int32ToPgInt4(i int32) pgtype.Int4 {
	return pgtype.Int4{Int32: i, Valid: true}
}

// Int32PtrToPgInt4 converts *int32 to pgtype.Int4
func Int32PtrToPgInt4(i *int32) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: *i, Valid: true}
}

// PgInt4ToInt32 converts pgtype.Int4 to int32 (returns 0 if invalid)
func PgInt4ToInt32(pi pgtype.Int4) int32 {
	if !pi.Valid {
		return 0
	}
	return pi.Int32
}

// PgInt4ToInt32Ptr converts pgtype.Int4 to *int32
func PgInt4ToInt32Ptr(pi pgtype.Int4) *int32 {
	if !pi.Valid {
		return nil
	}
	return &pi.Int32
}

// BoolToPgBool converts bool to pgtype.Bool
func BoolToPgBool(b bool) pgtype.Bool {
	return pgtype.Bool{Bool: b, Valid: true}
}

// BoolPtrToPgBool converts *bool to pgtype.Bool
func BoolPtrToPgBool(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{Valid: false}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}

// PgBoolToBool converts pgtype.Bool to bool (returns false if invalid)
func PgBoolToBool(pb pgtype.Bool) bool {
	if !pb.Valid {
		return false
	}
	return pb.Bool
}

// PgBoolToBoolPtr converts pgtype.Bool to *bool
func PgBoolToBoolPtr(pb pgtype.Bool) *bool {
	if !pb.Valid {
		return nil
	}
	return &pb.Bool
}

// TimeToPgTimestamptz converts time.Time to pgtype.Timestamptz
func TimeToPgTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// TimePtrToPgTimestamptz converts *time.Time to pgtype.Timestamptz
func TimePtrToPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil || t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// PgTimestamptzToTime converts pgtype.Timestamptz to time.Time
func PgTimestamptzToTime(pt pgtype.Timestamptz) time.Time {
	if !pt.Valid {
		return time.Time{}
	}
	return pt.Time
}

// PgTimestamptzToTimePtr converts pgtype.Timestamptz to *time.Time
func PgTimestamptzToTimePtr(pt pgtype.Timestamptz) *time.Time {
	if !pt.Valid {
		return nil
	}
	return &pt.Time
}
