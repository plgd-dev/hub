import ReactSelect, { components } from 'react-select'

const DropdownIndicator = props => {
  const icon = props.selectProps.menuIsOpen
    ? 'fa-chevron-up'
    : 'fa-chevron-down'
  return (
    <components.DropdownIndicator {...props}>
      <i className={`fas ${icon}`} />
    </components.DropdownIndicator>
  )
}

export const Select = props => {
  return (
    <ReactSelect
      {...props}
      classNamePrefix="select"
      components={{ DropdownIndicator }}
    />
  )
}
