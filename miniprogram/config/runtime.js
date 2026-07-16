// Build/deployment pipelines may replace trial/release with their HTTPS API
// origins. Keep release empty in source control so a production package can
// never silently fall back to a development address.
module.exports = {
  develop: "http://10.1.100.82:8080/api",
  trial: "",
  release: ""
}
