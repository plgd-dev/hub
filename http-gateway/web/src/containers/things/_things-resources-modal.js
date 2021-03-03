import { useEffect, useState, useRef } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'

import { Editor } from '@/components/editor'
import { Select } from '@/components/select'
import { Button } from '@/components/button'
import { Badge } from '@/components/badge'
import { Label } from '@/components/label'
import { Modal } from '@/components/modal'

import { resourceModalTypes } from './constants'
import { messages as t } from './things-i18n'

const NOOP = () => {}
const { CREATE_RESOURCE, UPDATE_RESOURCE } = resourceModalTypes

export const ThingsResourcesModal = ({
  data,
  deviceId,
  resourceData,
  onClose,
  retrieving,
  fetchResource,
  isDeviceOnline,
  loading,
  updateResource,
  createResource,
  type,
}) => {
  const { formatMessage: _ } = useIntl()
  const editor = useRef()
  const [jsonData, setJsonData] = useState(null)
  const [interfaceJsonError, setInterfaceJsonError] = useState(false)

  const disabled = retrieving || loading
  const isUpdateModal = type === UPDATE_RESOURCE
  const updateLabel = !loading ? _(t.update) : _(t.updating)
  const createLabel = !loading ? _(t.create) : _(t.creating)
  const initialInterfaceValue = { value: '', label: _(t.deviceInterfaces) }
  const [selectedInterface, setSelectedInterface] = useState(
    initialInterfaceValue
  )

  useEffect(
    () => {
      setJsonData(resourceData)

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

  const handleRetrieve = () => {
    fetchResource({
      href: data?.href,
      currentInterface: selectedInterface.value,
    })
  }

  const handleSubmit = () => {
    const params = {
      href: data?.href,
      currentInterface: selectedInterface.value,
    }

    if (isUpdateModal) {
      updateResource(params, jsonData)
    } else {
      createResource(params, jsonData)
    }
  }

  const handleCleanup = () => {
    setSelectedInterface(initialInterfaceValue)
    setJsonData(null)
  }

  const handleOnEditorChange = json => {
    if (json) {
      setJsonData(json)
    }
  }

  const handleOnEditorError = error => setInterfaceJsonError(error.length > 0)

  const renderBody = () => {
    return (
      <>
        <Label title={_(t.deviceId)} inline>
          {deviceId}
        </Label>
        <Label title={isUpdateModal ? _(t.types) : _(t.supportedTypes)} inline>
          <div className="align-items-end badges-box-vertical">
            {data?.types?.map?.(type => <Badge key={type}>{type}</Badge>) ||
              '-'}
          </div>
        </Label>

        {isUpdateModal && (
          <Label title={_(t.interfaces)} inline>
            <div className="align-items-end badges-box-vertical">
              {data?.interfaces?.map?.(ifs => <Badge key={ifs}>{ifs}</Badge>) ||
                '-'}
            </div>
          </Label>
        )}

        <div className="m-t-20 m-b-0">
          {jsonData && (
            <Editor
              json={jsonData}
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
        {isUpdateModal ? (
          <Select
            isDisabled={disabled || !isDeviceOnline}
            value={selectedInterface}
            onChange={setSelectedInterface}
            options={interfaces}
          />
        ) : (
          <div />
        )}

        <div>
          {isUpdateModal && (
            <Button
              variant="secondary"
              onClick={handleRetrieve}
              loading={retrieving}
              disabled={disabled}
            >
              {!retrieving ? _(t.retrieve) : _(t.retrieving)}
            </Button>
          )}

          <Button
            variant="primary"
            onClick={handleSubmit}
            loading={loading}
            disabled={disabled || interfaceJsonError}
          >
            {isUpdateModal ? updateLabel : createLabel}
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

ThingsResourcesModal.propTypes = {
  onClose: PropTypes.func,
  data: PropTypes.shape({
    href: PropTypes.string.isRequired,
    types: PropTypes.arrayOf(PropTypes.string),
    interfaces: PropTypes.arrayOf(PropTypes.string),
  }),
  deviceId: PropTypes.string,
  resourceData: PropTypes.object,
  retrieving: PropTypes.bool.isRequired,
  fetchResource: PropTypes.func.isRequired,
  loading: PropTypes.bool.isRequired,
  updateResource: PropTypes.func.isRequired,
  createResource: PropTypes.func.isRequired,
  isDeviceOnline: PropTypes.bool.isRequired,
  type: PropTypes.oneOf([CREATE_RESOURCE, UPDATE_RESOURCE]),
}

ThingsResourcesModal.defaultProps = {
  onClose: NOOP,
  data: null,
  deviceId: null,
  resourceData: null,
  type: UPDATE_RESOURCE,
}
