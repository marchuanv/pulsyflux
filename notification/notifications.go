package notification

type Notification string

const HTTP_CLIENT_RESPONSE_RECEVIED Notification = "04c3599b-ce2f-4a68-ba79-54af72e0e93e"
const HTTP_CLIENT_RESPONSE_ERROR Notification = "f4eb463b-06c5-4407-b6a8-a71f028e13e2"

const CONNECTION_CREATE Notification = "f11b4dc0-1f7d-43c2-94bb-33955d64c5d4"
const CONNECTION_OPEN Notification = "252706d8-48d4-4baa-9853-70b15d232f7d"
const CONNECTION_CLOSE Notification = "21938655-3a71-464a-b2db-3e7804f294b7"

const CONNECTION_CREATED Notification = "a0824a0c-4c2e-4ea1-b300-9067b086cf54"
const CONNECTION_OPENED Notification = "b022d911-c395-4894-b808-ef608485062f"
const CONNECTION_CLOSED Notification = "049188c2-d9ab-47dd-b632-1429f5e2a6b4"

const CONNECTION_ERROR Notification = "3cf7c28a-7c47-4f6e-b1fb-9e6326bd04d9"

const HTTP_SERVER_CREATE Notification = "476a02ea-9c5f-4275-afd1-f845da212bc3"
const HTTP_SERVER_START Notification = "616addea-1f09-44d6-94e5-b02a2a6b3b1c"
const HTTP_SERVER_STOP Notification = "27c1c919-b357-4ff1-89a2-ac02f00c49a5"

const HTTP_SERVER_CREATED Notification = "066e0a18-96c1-47fe-b28e-d30c97656650"
const HTTP_SERVER_STARTED Notification = "62804e10-3f8f-4ef4-b41b-fa91ee130cee"
const HTTP_SERVER_STOPPED Notification = "52f47d00-8625-4700-aec9-3f493e16d1e5"

const HTTP_SERVER_ERROR Notification = "772d5c6c-3604-4991-9c72-dd9200efb3f1"

const HTTP_SERVER_NO_RESPONSE Notification = "3d0ee0b0-70c2-4970-bd72-c8ae16d8339f"
const HTTP_SERVER_RESPONSE_RECEVIED Notification = "65b26bd3-b699-4fb4-930e-3b4283d69e9f"
const HTTP_SERVER_RESPONSE_ERROR Notification = "6705452a-e7c1-40fc-9261-737ce50a861b"

const HTTP_SERVER_REQUIRE_ADDRESS Notification = "b6b86ba2-6f4e-439b-9502-e52d3da48b21"
