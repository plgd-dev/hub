import { useEffect, useState } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'

import { Select } from '@/components/select'
import { Button } from '@/components/button'
import { Badge } from '@/components/badge'
import { Label } from '@/components/label'
import { Modal } from '@/components/modal'

import { messages as t } from './things-i18n'

const NOOP = () => {}

export const ThingsResourcesUpdateModal = ({
  data,
  resourceData,
  onClose,
  retrieving,
  fetchResource,
  isDeviceOnline,
}) => {
  const { formatMessage: _ } = useIntl()
  const [updating, setUpdating] = useState(false)
  const [selectedIntefaceData, setSelectedInterfaceData] = useState(null)

  const initialInterfaceValue = { value: '', label: _(t.deviceInterfaces) }
  const [selectedInterface, setSelectedInterface] = useState(
    initialInterfaceValue
  )

  useEffect(
    () => {
      setSelectedInterfaceData(resourceData)
    },
    [resourceData]
  )

  const disabled = retrieving || updating

  const renderBody = () => {
    return (
      <>
        <Label title={_(t.deviceId)} inline>
          {data?.di}
        </Label>
        <Label title={_(t.types)} inline>
          <div className="align-items-end badges-box-vertical">
            {data?.types?.map?.(type => <Badge key={type}>{type}</Badge>) ||
              '-'}
          </div>
        </Label>
        <Label title={_(t.interfaces)} inline>
          <div className="align-items-end badges-box-vertical">
            {data?.interfaces?.map?.(ifs => <Badge key={ifs}>{ifs}</Badge>) ||
              '-'}
          </div>
        </Label>
        <pre className="m-t-20 m-b-0">
          {selectedIntefaceData &&
            JSON.stringify(selectedIntefaceData, null, 2)}
        </pre>
      </>
    )
  }

  const renderFooter = () => {
    const interfaces =
      data?.interfaces?.map?.(ifs => ({ value: ifs, label: ifs })) || []
    interfaces.unshift(initialInterfaceValue)

    return (
      <div className="w-100 d-flex justify-content-between align-items-center">
        <Select
          isDisabled={disabled || !isDeviceOnline}
          value={selectedInterface}
          onChange={setSelectedInterface}
          options={interfaces}
        />
        <div>
          <Button
            variant="secondary"
            onClick={handleRetrieve}
            loading={retrieving}
            disabled={disabled}
          >
            {!retrieving ? _(t.retrieve) : _(t.retrieving)}
          </Button>
          <Button
            variant="primary"
            onClick={() => setUpdating(true)}
            loading={updating}
            disabled={disabled || true} // temporary disabled, next user story will implement the update
          >
            {!updating ? _(t.update) : _(t.updating)}
          </Button>
        </div>
      </div>
    )
  }

  const handleRetrieve = () => {
    fetchResource({
      di: data?.di,
      href: data?.href,
      currentInterface: selectedInterface.value,
    })
  }

  const handleCleanup = () => {
    setSelectedInterface(initialInterfaceValue)
    setSelectedInterfaceData(null)
  }

  return (
    <Modal
      show={!!data}
      onClose={!disabled ? onClose : NOOP}
      title={data?.href}
      renderBody={renderBody}
      renderFooter={renderFooter}
      onExited={handleCleanup}
      closeButton={!disabled}
    />
  )
}

ThingsResourcesUpdateModal.propTypes = {
  onClose: PropTypes.func,
  data: PropTypes.shape({
    di: PropTypes.string.isRequired,
    href: PropTypes.string.isRequired,
    types: PropTypes.arrayOf(PropTypes.string),
    interfaces: PropTypes.arrayOf(PropTypes.string),
  }),
  resourceData: PropTypes.object,
  retrieving: PropTypes.bool.isRequired,
  fetchResource: PropTypes.func.isRequired,
  isDeviceOnline: PropTypes.bool.isRequired,
}

ThingsResourcesUpdateModal.defaultProps = {
  onClose: NOOP,
  data: null,
  resourceData: null,
}
