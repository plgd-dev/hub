import React, { FC, useEffect, useMemo, useState } from 'react'
import { generatePath, useNavigate, useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'
import omit from 'lodash/omit'

import { useIsMounted, WellKnownConfigType } from '@shared-ui/common/hooks'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'
import { DeleteModal } from '@shared-ui/components/Atomic/Modal'
import { security } from '@shared-ui/common/services'
import DevicesResourcesModal from '@shared-ui/components/Organisms/DevicesResourcesModal'
import {
    DevicesDetailsResourceModalData,
    DevicesResourcesModalParamsType,
} from '@shared-ui/components/Organisms/DevicesResourcesModal/DevicesResourcesModal.types'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import { Property } from '@shared-ui/components/Organisms/GeneratedResourceForm/GeneratedResourceForm.types'

import { Props } from './Tab2.types'
import DevicesResources from '@/containers/Devices/Resources/DevicesResources'
import { createDevicesResourceApi, deleteDevicesResourceApi, getDevicesResourcesApi, updateDevicesResourceApi } from '@/containers/Devices/rest'
import { defaultNewResource, knownResourceHref, resourceModalTypes } from '@/containers/Devices/constants'
import {
    handleCreateResourceErrors,
    handleDeleteResourceErrors,
    handleUpdateResourceErrors,
    hasGeneratedResourcesForm,
    isErrorOnlyWarning,
} from '@/containers/Devices/utils'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { isNotificationActive, toggleActiveNotification } from '@/containers/Devices/slice'
import { deviceResourceUpdateListener } from '@/containers/Devices/websockets'
import { createResourceNotificationId } from '@/containers/PendingCommands/utils'
import notificationId from '@/notificationId'
import { pages } from '@/routes'
import testId from '@/testId'

const Tab2: FC<Props> = (props) => {
    const { deviceStatus, deviceName, isOnline, isActiveTab, isUnregistered, loading, resourcesData, loadingResources, refreshResources } = props
    const { id: routerId, ...others } = useParams()
    const id = routerId || ''
    const hrefParam = others['*'] || ''

    const [resourceModalData, setResourceModalData] = useState<DevicesDetailsResourceModalData | undefined>(undefined)
    const [loadingResource, setLoadingResource] = useState(false)
    const [savingResource, setSavingResource] = useState(false)
    const [deleteResourceHref, setDeleteResourceHref] = useState<string>('')
    const [resourceModal, setResourceModal] = useState(false)
    const [ttlHasError, setTtlHasError] = useState(false)

    const resources = useMemo(() => resourcesData?.[0]?.resources || [], [resourcesData])
    const { formatMessage: _ } = useIntl()
    const isMounted = useIsMounted()
    const navigate = useNavigate()

    const wellKnownConfig = security.getWellKnownConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }
    const [ttl, setTtl] = useState(wellKnownConfig?.defaultCommandTimeToLive || 0)

    const generatedResourcesForm = useMemo(() => hasGeneratedResourcesForm(resources), [resources])

    const loadFormData = async () => {
        try {
            const { data: resourceData } = await getDevicesResourcesApi({
                deviceId: id,
                href: knownResourceHref.WELL_KNOW_WOT,
                currentInterface: '',
            })

            return resourceData.data.content
        } catch (error) {
            console.error(error)
            if (error) {
                Notification.error(
                    { title: _(t.resourceGetKnowConfErrorTitle), message: _(t.resourceGetKnowConfErrorMessage) },
                    {
                        notificationId: notificationId.HUB_DEVICES_DETAILS_PAGE_TAB2_GET_RESOURCE,
                    }
                )
            }
        }
    }

    // Open the resource modal when href is present
    useEffect(
        () => {
            if (hrefParam && !loading && !loadingResources) {
                openUpdateModal({ href: `/${hrefParam}` }).then()
            }
        },
        [hrefParam, loading, loadingResources] // eslint-disable-line
    )

    // Fetches the resources supported types and sets its values to the modal data, which opens the modal.
    const openCreateModal = async (href: string) => {
        // If there is already a fetch for a resource, disable the next attempt for a fetch until the previous fetch finishes
        if (loadingResource) {
            return
        }

        setLoadingResource(true)

        try {
            const { data: deviceData } = await getDevicesResourcesApi({
                deviceId: id,
                href,
            })
            const supportedTypes = deviceData?.data?.content?.rts || []

            if (isMounted.current) {
                setLoadingResource(false)

                // Setting the data and opening the modal
                setResourceModalData({
                    data: {
                        href,
                        types: supportedTypes,
                    },
                    resourceData: {
                        ...defaultNewResource,
                        rt: supportedTypes,
                    },
                    formProperties: false,
                    type: resourceModalTypes.CREATE_RESOURCE,
                })
                setResourceModal(true)
            }
        } catch (error) {
            if (error && isMounted.current) {
                setLoadingResource(false)
                Notification.error(
                    {
                        title: _(t.resourceRetrieveError),
                        message: getApiErrorMessage(error),
                    },
                    {
                        notificationId: notificationId.HUB_DEVICES_DETAILS_PAGE_TAB2_OPEN_CREATE_MODAL,
                    }
                )
            }
        }
    }

    const openDeleteModal = (href: string) => {
        setDeleteResourceHref(href)
    }

    // Fetches the resource and sets its values to the modal data, which opens the modal.
    const openUpdateModal = async ({ href, currentInterface = '' }: { href: string; currentInterface?: string }) => {
        // If there is already a fetch for a resource, disable the next attempt for a fetch until the previous fetch finishes
        if (loadingResource) {
            return
        }

        setLoadingResource(true)

        try {
            const { data: resourceData } = await getDevicesResourcesApi({
                deviceId: id,
                href,
                currentInterface,
            })

            let formProperties: Property | false = false

            if (generatedResourcesForm) {
                const generatedFormResourceData = await loadFormData()
                formProperties = href === knownResourceHref.WELL_KNOW_WOT ? generatedFormResourceData : generatedFormResourceData?.properties[href]
            }

            omit(resourceData, ['data.content.if', 'data.content.rt'])

            if (isMounted.current) {
                setLoadingResource(false)

                // Retrieve the types and interfaces of this resource
                const { resourceTypes: types = [], interfaces = [] } = resources?.find?.((link: { href: string }) => link.href === href) || {}

                // Setting the data and opening the modal
                setResourceModalData({
                    data: {
                        href,
                        types,
                        interfaces,
                    },
                    resourceData,
                    formProperties,
                })
                setResourceModal(true)
                navigate(`${generatePath(pages.DEVICES.DETAIL.LINK, { id, tab: pages.DEVICES.DETAIL.TABS[1], section: '' })}${href}`)
            }
        } catch (error) {
            if (error && isMounted.current) {
                setLoadingResource(false)
                Notification.error(
                    {
                        title: _(t.resourceRetrieveError),
                        message: getApiErrorMessage(error),
                    },
                    {
                        notificationId: notificationId.HUB_DEVICES_DETAILS_PAGE_TAB2_OPEN_UPDATE_MODAL,
                    }
                )
                navigate(generatePath(pages.DEVICES.DETAIL.LINK, { id, tab: pages.DEVICES.DETAIL.TABS[1], section: '' }))
            }
        }
    }

    const createResource = async ({ href, currentInterface = '' }: DevicesResourcesModalParamsType, resourceDataCreate: object) => {
        setSavingResource(true)

        try {
            const ret = await createDevicesResourceApi({ deviceId: id, href, currentInterface, ttl }, resourceDataCreate)
            const { auditContext } = ret.data.data

            if (isMounted.current) {
                Notification.success(
                    { title: _(t.resourceCreateSuccess), message: _(t.resourceWasCreated) },
                    {
                        toastId: createResourceNotificationId(id, href, auditContext?.correlationId, auditContext?.userId),
                        notificationId: notificationId.HUB_DEVICES_DETAILS_PAGE_TAB2_CREATE_RESOURCE,
                    }
                )
                setResourceModalData(undefined) // close modal
                setSavingResource(false)
                refreshResources()
            }
        } catch (error) {
            if (error && isMounted.current) {
                handleCreateResourceErrors(error, { id, href }, _)
                setSavingResource(false)
            }
        }
    }

    const updateResource = async ({ href, currentInterface = '' }: DevicesResourcesModalParamsType, resourceDataUpdate: any) => {
        setSavingResource(true)

        try {
            const ret = await updateDevicesResourceApi({ deviceId: id, href, currentInterface, ttl }, resourceDataUpdate)
            const { auditContext } = ret.data.data

            if (isMounted.current) {
                Notification.success(
                    { title: _(t.resourceUpdateSuccess), message: _(t.resourceWasUpdated) },
                    {
                        toastId: createResourceNotificationId(id, href, auditContext?.correlationId, auditContext?.userId),
                        notificationId: notificationId.HUB_DEVICES_DETAILS_PAGE_TAB2_UPDATE_RESOURCE,
                    }
                )

                handleCloseUpdateModal()
                setSavingResource(false)
            }
        } catch (error) {
            if (error && isMounted.current) {
                handleUpdateResourceErrors(error, { id, href }, _)
                isErrorOnlyWarning(error) && handleCloseUpdateModal()
                setSavingResource(false)
            }
        }
    }

    const deleteResource = async () => {
        setLoadingResource(true)

        try {
            const ret = await deleteDevicesResourceApi({
                deviceId: id,
                href: deleteResourceHref,
                ttl,
            })
            const { auditContext } = ret.data.data

            if (isMounted.current) {
                Notification.success(
                    { title: _(t.resourceDeleteSuccess), message: _(t.resourceWasDeleted) },
                    {
                        toastId: createResourceNotificationId(id, deleteResourceHref, auditContext?.correlationId, auditContext?.userId),
                        notificationId: notificationId.HUB_DEVICES_DETAILS_PAGE_TAB2_DELETE_RESOURCE,
                    }
                )

                setLoadingResource(false)
                closeDeleteModal()
            }
        } catch (error) {
            if (error && isMounted.current) {
                handleDeleteResourceErrors(error, { id, href: deleteResourceHref }, _)
                setLoadingResource(false)
                closeDeleteModal()
            }
        }
    }

    const handleCloseUpdateModal = () => {
        setResourceModalData(undefined)

        navigate(generatePath(pages.DEVICES.DETAIL.LINK, { id, tab: pages.DEVICES.DETAIL.TABS[1], section: '' }))
    }

    const closeDeleteModal = () => {
        setDeleteResourceHref('')
    }

    return (
        <>
            <Loadable condition={!!resourcesData}>
                <DevicesResources
                    data={resources}
                    deviceStatus={deviceStatus}
                    isActiveTab={isActiveTab}
                    loading={loadingResource}
                    onCreate={openCreateModal}
                    onDelete={openDeleteModal}
                    onUpdate={openUpdateModal}
                    // pageSize={{ width: pageSize.width, height: pageSize.height }} // tree switch
                />
            </Loadable>
            <DevicesResourcesModal
                {...resourceModalData}
                confirmDisabled={ttlHasError}
                createResource={createResource}
                dataTestId={testId.devices.detail.resources.updateModal}
                deviceId={id}
                deviceName={deviceName}
                deviceResourceUpdateListener={deviceResourceUpdateListener}
                fetchResource={openUpdateModal}
                generatedResourcesForm={generatedResourcesForm}
                i18n={{
                    advancedView: _(g.advancedView),
                    close: _(t.close),
                    commandTimeout: _(t.commandTimeout),
                    compactView: _(g.compactView),
                    content: _(t.content),
                    create: _(t.create),
                    creating: _(t.creating),
                    deviceId: _(t.deviceId),
                    fullView: _(g.fullView),
                    interfaces: _(t.interfaces),
                    invalidNumber: _(g.invalidNumber),
                    maxValue: (field: string, length: number) => _(g.maxValue, { field, length }),
                    minValue: (field: string, length: number) => _(g.minValue, { field, length }),
                    notifications: _(t.notifications),
                    off: _(t.off),
                    on: _(t.on),
                    resourceInterfaces: _(t.resourceInterfaces),
                    retrieve: _(t.retrieve),
                    retrieving: _(t.retrieving),
                    supportedTypes: _(t.supportedTypes),
                    types: _(t.types),
                    update: _(t.update),
                    updating: _(t.updating),
                }}
                isDeviceOnline={isOnline}
                isNotificationActive={isNotificationActive}
                isUnregistered={isUnregistered}
                loading={savingResource}
                onClose={handleCloseUpdateModal}
                retrieving={loadingResource}
                show={resourceModal}
                toggleActiveNotification={toggleActiveNotification}
                ttlControl={
                    <TimeoutControl
                        defaultTtlValue={wellKnownConfig?.defaultCommandTimeToLive || 0}
                        defaultValue={ttl}
                        disabled={loadingResource || savingResource}
                        i18n={{
                            default: _(t.default),
                            duration: _(t.duration),
                            placeholder: _(t.placeholder),
                            unit: _(t.unit),
                        }}
                        onChange={setTtl}
                        onTtlHasError={setTtlHasError}
                        size='small'
                        ttlHasError={ttlHasError}
                        unitMenuPortalTarget={document.body}
                    />
                }
                updateResource={updateResource}
            />
            <DeleteModal
                dataTestId={testId.devices.detail.resources.deleteModal}
                deleteInformation={[
                    {
                        label: _(t.commandTimeout),
                        value: (
                            <TimeoutControl
                                defaultTtlValue={wellKnownConfig?.defaultCommandTimeToLive || 0}
                                defaultValue={ttl}
                                disabled={loadingResource}
                                i18n={{
                                    default: _(t.default),
                                    duration: _(t.duration),
                                    placeholder: _(t.placeholder),
                                    unit: _(t.unit),
                                }}
                                onChange={setTtl}
                                onTtlHasError={setTtlHasError}
                                ttlHasError={ttlHasError}
                            />
                        ),
                    },
                ]}
                footerActions={[
                    {
                        label: _(t.cancel),
                        onClick: closeDeleteModal,
                        variant: 'tertiary',
                    },
                    {
                        label: _(t.delete),
                        onClick: deleteResource,
                        variant: 'primary',
                    },
                ]}
                onClose={closeDeleteModal}
                show={!!deleteResourceHref}
                subTitle={_(t.deleteResourceMessageSubtitle)}
                title={_(t.deleteResourceMessage)}
            />
        </>
    )
}

Tab2.displayName = 'Tab2'
export default Tab2
