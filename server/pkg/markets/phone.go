package markets

import "strings"

func NormalizePhone(country, phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	phone = strings.TrimPrefix(phone, "+")

	if strings.HasPrefix(phone, "00") && len(phone) > 2 {
		phone = phone[2:]
	}

	prefix := CallingCode(country)
	if prefix == "" || phone == "" {
		return phone
	}

	if strings.HasPrefix(phone, prefix) {
		return phone
	}

	if strings.HasPrefix(phone, "0") && len(phone) > 1 {
		return prefix + phone[1:]
	}

	return prefix + phone
}
