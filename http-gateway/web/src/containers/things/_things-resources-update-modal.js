import { useEffect, useState, useRef } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'

import { Editor } from '@/components/editor'
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
  updating,
  updateResource,
}) => {
  const { formatMessage: _ } = useIntl()
  const editor = useRef()
  const [selectedIntefaceData, setSelectedInterfaceData] = useState(null)
  const [interfaceJsonError, setInterfaceJsonError] = useState(false)

  const initialInterfaceValue = { value: '', label: _(t.deviceInterfaces) }
  const [selectedInterface, setSelectedInterface] = useState(
    initialInterfaceValue
  )

  useEffect(
    () => {
      setSelectedInterfaceData(resourceData)

      if (resourceData) {
        // Set the retrieved JSON object to the editor
        if (typeof resourceData === 'object') {
          editor?.current?.set(resourceData)
        } else if (typeof resourceData === 'string') {
          editor?.current?.setText(resourceData)
        }
      }
    },
    [resourceData]
  )

  const disabled = retrieving || updating

  const handleRetrieve = () => {
    fetchResource({
      href: data?.href,
      currentInterface: selectedInterface.value,
    })
  }

  const handleUpdate = () => {
    updateResource(
      {
        href: data?.href,
        currentInterface: selectedInterface.value,
      },
      selectedIntefaceData
    )
  }

  const handleCleanup = () => {
    setSelectedInterface(initialInterfaceValue)
    setSelectedInterfaceData(null)
  }

  const handleOnEditorChange = json => {
    if (json) {
      setSelectedInterfaceData(json)
    }
  }

  const handleOnEditorError = error => setInterfaceJsonError(error.length > 0)

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
        <div className="m-t-20 m-b-0">
          {selectedIntefaceData && (
            <Editor
              json={selectedIntefaceData}
              onChange={handleOnEditorChange}
              onError={handleOnEditorError}
              editorRef={node => {
                editor.current = node
              }}
              disabled={disabled}
            />
          )}
        </div>
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
            onClick={handleUpdate}
            loading={updating}
            disabled={disabled || interfaceJsonError}
          >
            {!updating ? _(t.update) : _(t.updating)}
          </Button>
        </div>
      </div>
    )
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
  updating: PropTypes.bool.isRequired,
  updateResource: PropTypes.func.isRequired,
  isDeviceOnline: PropTypes.bool.isRequired,
}

ThingsResourcesUpdateModal.defaultProps = {
  onClose: NOOP,
  data: null,
  resourceData: null,
}
