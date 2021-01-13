exports.format = (messages) => {
  let array = []
  Object.keys(messages).forEach(function (key) {
    array.push({
      id: key,
      defaultMessage: messages[key].defaultMessage,
    })
  })
  return array
}