export const copyToClipboard = text => {
  if (document.execCommand) {
    const textField = document.createElement('textarea')
    textField.innerText = text
    document.body.appendChild(textField)
    textField.select()
    document.execCommand('copy')
    textField.remove()

    return true
  }

  return false
}
