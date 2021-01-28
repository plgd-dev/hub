import BBadge from 'react-bootstrap/Badge'

export const Badge = ({ children, ...rest }) => {
  return (
    <BBadge pill {...rest}>
      {children}
    </BBadge>
  )
}
