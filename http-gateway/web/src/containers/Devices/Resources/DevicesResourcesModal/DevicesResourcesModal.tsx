import { useEffect, useState, useRef, FC } from 'react'
import { useIntl } from 'react-intl'

import Editor from '@shared-ui/components/new/Editor'
import Select from '@shared-ui/components/new/Select'
import Button from '@shared-ui/components/new/Button'
import Badge from '@shared-ui/components/new/Badge'
import Label from '@shared-ui/components/new/Label'
import Modal from '@shared-ui/components/new/Modal'

import DevicesResourcesModalNotifications from '../DevicesResourcesModalNotifications'
import { resourceModalTypes } from '../../constants'
import { messages as t } from '../../Devices.i18n'
import { Props, defaultProps } from './DevicesResourcesModal.types'

const NOOP = () => {}
const { UPDATE_RESOURCE } = resourceModalTypes

const DevicesResourcesModal: FC<Props> = props => {
  const {
    data,
    deviceId,
    deviceName,
    resourceData,
    onClose,
    retrieving,
    fetchResource,
    isDeviceOnline,
    isUnregistered,
    loading,
    updateResource,
    createResource,
    type,
    ttlControl,
    confirmDisabled,
  } = { ...defaultProps, ...props }
  const { formatMessage: _ } = useIntl()
  const editor = useRef()
  const [jsonData, setJsonData] = useState<object | undefined>(undefined)
  const [interfaceJsonError, setInterfaceJsonError] = useState(false)

  const disabled = retrieving || loading
  const isUpdateModal = type === UPDATE_RESOURCE
  const updateLabel = !loading ? _(t.update) : _(t.updating)
  const createLabel = !loading ? _(t.create) : _(t.creating)
  const initialInterfaceValue = { value: '', label: _(t.resourceInterfaces) }
  const [selectedInterface, setSelectedInterface] = useState(
    initialInterfaceValue
  )

  useEffect(() => {
    setJsonData(resourceData)

    if (resourceData) {
      // Set the retrieved JSON object to the editor
      if (typeof resourceData === 'object') {
        // @ts-ignore
        editor?.current?.set(resourceData)
      } else if (typeof resourceData === 'string') {
        // @ts-ignore
        editor?.current?.setText(resourceData)
      }
    }
  }, [resourceData])

  const handleRetrieve = () => {
    fetchResource({
      href: data?.href as string,
      currentInterface: selectedInterface.value,
    })
  }

  const handleSubmit = () => {
    const params = {
      href: data?.href as string,
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
    setJsonData(undefined)
  }

  const handleOnEditorChange = (json: object) => {
    json && setJsonData(json)
  }

  const handleOnEditorError = (error: any) =>
    setInterfaceJsonError(error.length > 0)

  const renderBody = () => {
    return (
      <>
        {data && isUpdateModal && (
          <Label title="" inline>
            <DevicesResourcesModalNotifications
              deviceId={deviceId as string}
              deviceName={deviceName}
              href={data?.href as string}
              isUnregistered={isUnregistered}
            />
          </Label>
        )}

        <Label title={_(t.deviceId)} inline>
          {deviceId}
        </Label>
        <Label title={isUpdateModal ? _(t.types) : _(t.supportedTypes)} inline>
          <div className="align-items-end badges-box-vertical">
            {data?.types?.map?.(_type => <Badge key={_type}>{_type}</Badge>) ||
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

        {ttlControl}

        <div className="m-t-20 m-b-0">
          {jsonData && (
            <Editor
              json={jsonData}
              onChange={handleOnEditorChange}
              onError={handleOnEditorError}
              editorRef={(node: any) => {
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
            disabled={disabled || interfaceJsonError || confirmDisabled}
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

DevicesResourcesModal.displayName = 'DevicesResourcesModal'
DevicesResourcesModal.defaultProps = defaultProps

export default DevicesResourcesModal
