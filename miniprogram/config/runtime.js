// Build/deployment pipelines may replace trial/release with their HTTPS API
// origins. Keep release empty in source control so a production package can
// never silently fall back to a development address.
module.exports = {
	develop: "http://127.0.0.1:8080/api",
	trial: "https://www.xueju.xyz/api",
	release: ""
}
