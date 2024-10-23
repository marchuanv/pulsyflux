package subscriptions

import "pulsyflux/msgbus"

const (
	HTTP msgbus.MsgSubId = "452aa68e-7f90-47aa-b475-9a6263c3cd4b"

	START_HTTP_SERVER           msgbus.MsgSubId = "8438288d-c4fe-411f-8f35-7db4e887e497"
	HTTP_SERVER_STARTED         msgbus.MsgSubId = "d1f34c8c-94c3-43b8-82ef-bb9b61591560"
	FAILED_TO_START_HTTP_SERVER msgbus.MsgSubId = "f5e5b557-4fd3-418a-ab5c-7ddf2e3cd546"

	STOP_HTTP_SERVER           msgbus.MsgSubId = "a183e779-289b-4816-a0fa-d5eae11f42d5"
	FAILED_TO_STOP_HTTP_SERVER msgbus.MsgSubId = "1e58a866-8991-4746-920e-7bf71720c8b9"

	FAILED_TO_LISTEN_ON_HTTP_SERVER_PORT msgbus.MsgSubId = "460feb2d-2783-4046-b2dc-ad2769398bcd"

	HTTP_SERVER_RESPONSE         msgbus.MsgSubId = "69ef7396-9986-42a0-960f-509b73f371a2"
	HTTP_SERVER_SUCCESS_RESPONSE msgbus.MsgSubId = "61ca9656-b862-436e-8d06-42ce4f448632"
	HTTP_SERVER_ERROR_RESPONSE   msgbus.MsgSubId = "57dcd78d-49d0-4125-889a-96a783ab367e"
)
