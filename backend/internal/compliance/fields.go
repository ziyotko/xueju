package compliance

const ContentRiskMessage = "内容可能包含不适宜信息，请修改后重试"

type TextField string

const (
	FieldEventTitle   TextField = "ski_event_title"
	FieldEventRemark  TextField = "ski_event_remark"
	FieldUserNickname TextField = "user_nickname"
	FieldUserBio      TextField = "user_bio"
	FieldChatMessage  TextField = "chat_message"
	FieldReview       TextField = "review_content"
	FieldReport       TextField = "report_content"
)

type ModerationStatus string

const (
	ModerationPending  ModerationStatus = "pending"
	ModerationApproved ModerationStatus = "approved"
	ModerationRejected ModerationStatus = "rejected"
	ModerationHidden   ModerationStatus = "hidden"
)

type AdminAction string

const (
	ActionDisableUser   AdminAction = "disable_user"
	ActionDelistEvent   AdminAction = "delist_event"
	ActionHideMessage   AdminAction = "hide_message"
	ActionHideReview    AdminAction = "hide_review"
	ActionResolveReport AdminAction = "resolve_report"
)
