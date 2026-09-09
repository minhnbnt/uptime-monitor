package dto

type NotificationConfigRequest struct {
	FromDate   string
	ToDate     string
	DigestTime string
	Timezone   string
}

type NotificationConfigResponse struct {
	FromDate   string
	ToDate     string
	DigestTime string
	Timezone   string
}
