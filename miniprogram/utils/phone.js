function isValidPhone(value) {
  if (!value) return true
  return /^1[3-9]\d{9}$/.test(String(value).trim())
}

function maskPhone(value) {
  const phone = String(value || '').trim()
  if (!/^1[3-9]\d{9}$/.test(phone)) return ''
  return `${phone.slice(0, 3)}****${phone.slice(7)}`
}

module.exports = {
  isValidPhone,
  maskPhone
}
