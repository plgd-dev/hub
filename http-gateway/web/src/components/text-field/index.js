import Form from 'react-bootstrap/Form'

export const TextField = ({ value, ...rest }) => {
  return <Form.Control {...rest} type="text" value={value} />
}
