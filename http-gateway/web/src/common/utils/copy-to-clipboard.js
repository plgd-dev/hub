export const copyToClipboard = (text, certFormat = false) => {
  if (document.execCommand) {
    const textField = document.createElement('textarea')
    textField.innerText = certFormat
      ? JSON.stringify(text.replace(/<br>/g, '\n'))
      : text
    document.body.appendChild(textField)
    textField.select()
    document.execCommand('copy')
    textField.remove()

    return true
  }

  return false
}
