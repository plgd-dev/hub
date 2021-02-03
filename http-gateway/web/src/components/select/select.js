import ReactSelect, { components } from 'react-select'

const DropdownIndicator = props => {
  return (
    <components.DropdownIndicator {...props}>
      <i className="fas fa-chevron-down" />
    </components.DropdownIndicator>
  )
}

export const Select = props => {
  return <ReactSelect {...props} classNamePrefix="select" components={{ DropdownIndicator }} />
}
