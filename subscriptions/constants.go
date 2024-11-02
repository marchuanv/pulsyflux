package subscriptions

import "pulsyflux/msgbus"

const (
	HTTP msgbus.MsgSubId = "452aa68e-7f90-47aa-b475-9a6263c3cd4b"

	START_HTTP_SERVER           msgbus.MsgSubId = "8438288d-c4fe-411f-8f35-7db4e887e497"
	HTTP_SERVER_STARTED         msgbus.MsgSubId = "d1f34c8c-94c3-43b8-82ef-bb9b61591560"
	FAILED_TO_START_HTTP_SERVER msgbus.MsgSubId = "f5e5b557-4fd3-418a-ab5c-7ddf2e3cd546"
	RECEIVE_HTTP_SERVER_ADDRESS msgbus.MsgSubId = "416a9a28-a5df-404a-8cc3-6033029f7d40"
	INVALID_HTTP_SERVER_ADDRESS msgbus.MsgSubId = "604a9df0-a1fc-4c67-aff1-52eae6af2f29"

	STOP_HTTP_SERVER           msgbus.MsgSubId = "a183e779-289b-4816-a0fa-d5eae11f42d5"
	FAILED_TO_STOP_HTTP_SERVER msgbus.MsgSubId = "1e58a866-8991-4746-920e-7bf71720c8b9"

	FAILED_TO_LISTEN_ON_HTTP_SERVER_PORT msgbus.MsgSubId = "460feb2d-2783-4046-b2dc-ad2769398bcd"

	HTTP_SERVER_RESPONSE         msgbus.MsgSubId = "69ef7396-9986-42a0-960f-509b73f371a2"
	HTTP_SERVER_SUCCESS_RESPONSE msgbus.MsgSubId = "61ca9656-b862-436e-8d06-42ce4f448632"
	HTTP_SERVER_ERROR_RESPONSE   msgbus.MsgSubId = "57dcd78d-49d0-4125-889a-96a783ab367e"

	HTTP_REQUEST         msgbus.MsgSubId = "bfe776d5-76b5-4e69-a895-b1806968d4f8"
	INVALID_HTTP_REQUEST msgbus.MsgSubId = "17cc5f2e-d0ab-4c4e-933e-a8f0e92c3313"

	HTTP_REQUEST_METHOD         msgbus.MsgSubId = "5e607279-1446-4d46-894e-7cb47da87f09"
	INVALID_HTTP_REQUEST_METHOD msgbus.MsgSubId = "2ed32381-b69a-475b-bf4d-6e73c49e022f"

	REQUEST_PROTOCAL         msgbus.MsgSubId = "ed52a842-3984-4c14-afb7-ef19b87f91c2"
	INVALID_REQUEST_PROTOCAL msgbus.MsgSubId = "0b8f9616-dca3-4578-8ca7-50133510fd0f"

	HTTP_RESPONSE msgbus.MsgSubId = "0cf46747-16f5-4d83-b6b3-c7fafbe463a5"

	HTTP_REQUEST_SUCCESS_RESPONSE msgbus.MsgSubId = "0bb6a926-d417-4dda-9313-16ca96e521a6"
	HTTP_REQUEST_ERROR_RESPONSE   msgbus.MsgSubId = "2034b484-1058-47a6-ad45-2cb0e4494046"

	HTTP_REQUEST_ADDRESS         msgbus.MsgSubId = "3583ac70-d38c-4f15-b0f8-dbe4fd5bfd95"
	INVALID_HTTP_REQUEST_ADDRESS msgbus.MsgSubId = "5f61ea9d-e7e0-4a0b-8bab-7ab70c2f0eec"

	HTTP_REQUEST_PATH         msgbus.MsgSubId = "1e35713c-533b-423a-8aed-d04b9fcc07ad"
	INVALID_HTTP_REQUEST_PATH msgbus.MsgSubId = "4d4b5967-30b8-411a-8f05-040dc013c778"

	HTTP_REQUEST_DATA         msgbus.MsgSubId = "99a30dfa-3663-4572-8c33-15261e1b90a2"
	INVALID_HTTP_REQUEST_DATA msgbus.MsgSubId = "052a80a1-29f7-4088-a1bd-b116140be65a"
)
