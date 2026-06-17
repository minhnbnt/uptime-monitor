package dto

type NotificationConfigRequest struct {
	FromDate   string
	ToDate     string
	DigestTime string
}

type NotificationConfigResponse struct {
	FromDate   string
	ToDate     string
	DigestTime string
}
