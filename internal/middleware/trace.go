package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
)

var validRequestID = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,64}$`)

func Trace() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")

		if !validRequestID.MatchString(requestID) {
			requestID = generateRequestID()
		}

		c.Set("X-Request-ID", requestID)

		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		// ไม่บันทึก body หรือข้อมูลส่วนบุคคลของผู้ใช้
		status := c.Response().StatusCode()

		println(
			"request_id=" + requestID +
				" method=" + c.Method() +
				" path=" + c.Path() +
				" status=" + itoa(status) +
				" duration=" + duration.String(),
		)

		return err
	}
}

func generateRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000")))
	}

	return hex.EncodeToString(b[:])
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var buf [20]byte
	i := len(buf)

	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}

	return string(buf[i:])
}
