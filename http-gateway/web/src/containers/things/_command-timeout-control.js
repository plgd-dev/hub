import { useState } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'
import isFinite from 'lodash/isFinite'
import { time } from 'units-converter'
import classNames from 'classnames'

import { Select } from '@/components/select'
import { Label } from '@/components/label'
import { TextField } from '@/components/text-field'
import { commandTimeoutUnits } from './constants'
import { messages as t } from './things-i18n'

const { INFINITY, MS, NS } = commandTimeoutUnits

const MINIMAL_MS_VALUE = 100

const convertValueToNs = (value, unit) =>
  +time(value)
    .from(unit === INFINITY ? NS : unit)
    .to(NS)
    .value.toFixed(0)

const convertValueFromTo = (value, unitFrom, unitTo) =>
  time(value)
    .from(unitFrom === INFINITY ? NS : unitFrom)
    .to(unitTo === INFINITY ? NS : unitTo).value

const normalizeValue = value => +value.toFixed(5)

const hasError = (value, unit) => {
  const baseUnit = unit === INFINITY ? NS : unit

  const valueMs = time(value)
    .from(baseUnit)
    .to(MS).value

  if (valueMs < MINIMAL_MS_VALUE && value !== 0) {
    return true
  }

  return false
}

const convertAndNormalizeValueFromTo = (value, unitFrom, unitTo) =>
  normalizeValue(convertValueFromTo(value, unitFrom, unitTo))

export const CommanTimeoutControl = ({
  defaultValue,
  onChange,
  disabled,
  ttlHasError,
  onTtlHasError,
  isDelete,
}) => {
  const { formatMessage: _ } = useIntl()
  const [unit, setUnit] = useState(defaultValue === 0 ? INFINITY : MS)
  const [inputValue, setInputValue] = useState(
    convertAndNormalizeValueFromTo(defaultValue, NS, MS)
  )

  const units = Object.values(commandTimeoutUnits)
    .filter(unit => unit !== NS)
    .map(unit => ({
      value: unit,
      label: unit,
    }))

  const handleOnUnitChange = ({ value: unitValue }) => {
    const newInputValue =
      unitValue === INFINITY
        ? 0
        : convertAndNormalizeValueFromTo(inputValue, unit, unitValue)
    setInputValue(newInputValue)
    setUnit(unitValue)
    onChange(convertValueToNs(newInputValue, unitValue))

    if (unitValue === INFINITY) {
      onTtlHasError(false)
    }
  }

  const handleOnValueChange = event => {
    const onlyZerosRegex = /^0+$/
    const value = event.target.value.trim()
    const floatValue = parseFloat(value)
    const containsOneOrNoneDot = (value.match(/./g) || []).length <= 1
    const isValidNumber = isFinite(floatValue) || containsOneOrNoneDot
    const finiteFloatValue = isValidNumber ? value : 0
    const newInputValue = !!finiteFloatValue?.match?.(onlyZerosRegex)
      ? 0
      : finiteFloatValue

    if (value === '' || containsOneOrNoneDot || newInputValue >= 0) {
      setInputValue(newInputValue)
    }

    onTtlHasError(false)
  }

  const handleOnValueBlur = event => {
    const value = event.target.value.trim()
    const floatValue = parseFloat(value)

    if (floatValue === 0 || value === '') {
      onChange(0)
      // Change the dropdown to Infinity when provided 0 as value
      // setUnit(INFINITY)
    } else if (isFinite(floatValue) && floatValue > 0) {
      const newValue = normalizeValue(floatValue, unit)
      setInputValue(newValue)
      onChange(convertValueToNs(newValue, unit))

      if (hasError(newValue, unit)) {
        onTtlHasError(true)
      }
    }
  }

  return (
    <Label
      title={_(t.commandTimeout)}
      inline
      onClick={e => e.preventDefault()}
      className={classNames('command-timeout-label', {
        'delete-modal': isDelete,
      })}
    >
      <div className="ttl-label-content d-flex justify-content-end">
        <TextField
          className={classNames('ttl-value-input', { error: ttlHasError })}
          value={inputValue}
          onChange={handleOnValueChange}
          onBlur={handleOnValueBlur}
          placeholder="Infinity"
          disabled={disabled || unit === INFINITY}
        />
        <Select
          className="ttl-unit-dropdown"
          isDisabled={disabled}
          value={units.filter(option => option.value === unit)}
          onChange={handleOnUnitChange}
          options={units}
          styles={{
            control: ({ ...css }) => ({
              ...css,
              minWidth: '90px !important',
              borderTopLeftRadius: '0 !important',
              borderBottomLeftRadius: '0 !important',
            }),
          }}
        />
      </div>
      {ttlHasError && (
        <div className="error-message">
          {_(t.minimalValueIs, { minimalValue: '100ms' })}
        </div>
      )}
    </Label>
  )
}

CommanTimeoutControl.propTypes = {
  defaultValue: PropTypes.number.isRequired,
  onChange: PropTypes.func.isRequired,
  disabled: PropTypes.bool.isRequired,
  ttlHasError: PropTypes.bool.isRequired,
  onTtlHasError: PropTypes.func.isRequired,
  isDelete: PropTypes.bool,
}

CommanTimeoutControl.defaultProps = {
  isDelete: false,
}
