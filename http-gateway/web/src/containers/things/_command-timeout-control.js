import { useState } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'
import isFinite from 'lodash/isFinite'
import classNames from 'classnames'

import { Select } from '@/components/select'
import { Label } from '@/components/label'
import { TextField } from '@/components/text-field'
import { commandTimeoutUnits } from './constants'
import {
  convertValueToNs,
  hasCommandTimeoutError,
  convertAndNormalizeValueFromTo,
  normalizeToFixedFloatValue,
} from './utils'
import { messages as t } from './things-i18n'

const { INFINITE, MS, NS } = commandTimeoutUnits

export const CommanTimeoutControl = ({
  defaultValue,
  onChange,
  disabled,
  ttlHasError,
  onTtlHasError,
  isDelete,
}) => {
  const { formatMessage: _ } = useIntl()
  const [unit, setUnit] = useState(defaultValue === 0 ? INFINITE : MS)
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
      unitValue === INFINITE
        ? 0
        : convertAndNormalizeValueFromTo(inputValue, unit, unitValue)
    setInputValue(newInputValue)
    setUnit(unitValue)
    onChange(convertValueToNs(newInputValue, unitValue))

    if (unitValue === INFINITE) {
      onTtlHasError(false)
    }
  }

  const handleOnValueChange = event => {
    const onlyZerosRegex = /^0+$/
    const value = event.target.value.trim()
    const floatValue = parseFloat(value)
    const containsOneOrNoDot = (value.match(/./g) || []).length <= 1
    const isValidNumber = isFinite(floatValue) || containsOneOrNoDot
    const finiteFloatValue = isValidNumber ? value : 0
    const newInputValue = !!finiteFloatValue?.match?.(onlyZerosRegex)
      ? 0
      : finiteFloatValue

    if (value === '' || containsOneOrNoDot || newInputValue >= 0) {
      setInputValue(newInputValue)
    }

    onTtlHasError(false)
  }

  const handleOnValueBlur = event => {
    const value = event.target.value.trim()
    const floatValue = parseFloat(value)

    if (floatValue === 0 || value === '') {
      onChange(0)
    } else if (isFinite(floatValue) && floatValue > 0) {
      const newValue = normalizeToFixedFloatValue(floatValue, unit)
      setInputValue(newValue)
      onChange(convertValueToNs(newValue, unit))

      if (hasCommandTimeoutError(newValue, unit)) {
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
        {unit !== INFINITE && (
          <TextField
            className={classNames('ttl-value-input', { error: ttlHasError })}
            value={inputValue}
            onChange={handleOnValueChange}
            onBlur={handleOnValueBlur}
            placeholder="INFINITE"
            disabled={disabled || unit === INFINITE}
          />
        )}

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
  defaultValue: PropTypes.number,
  onChange: PropTypes.func.isRequired,
  disabled: PropTypes.bool.isRequired,
  ttlHasError: PropTypes.bool.isRequired,
  onTtlHasError: PropTypes.func.isRequired,
  isDelete: PropTypes.bool,
}

CommanTimeoutControl.defaultProps = {
  isDelete: false,
  defaultValue: 0,
}
