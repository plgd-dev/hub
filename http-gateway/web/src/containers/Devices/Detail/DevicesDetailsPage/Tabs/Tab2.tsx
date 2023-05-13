import React, { FC, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'
import { useResizeDetector } from 'react-resize-detector'
import omit from 'lodash/omit'

import { useIsMounted, WellKnownConfigType } from '@shared-ui/common/hooks'
import TimeoutControl from '@shared-ui/components/new/TimeoutControl'
import { DeleteModal } from '@shared-ui/components/new/Modal'
import { security } from '@shared-ui/common/services'
import DevicesResourcesModal from '@shared-ui/components/organisms/DevicesResourcesModal'
import { DevicesResourcesModalParamsType } from '@shared-ui/components/organisms/DevicesResourcesModal/DevicesResourcesModal.types'
import Notification from '@shared-ui/components/new/Notification/Toast'

import { Props } from './Tab2.types'
import DevicesResources from '@/containers/Devices/Resources/DevicesResources'
import { createDevicesResourceApi, deleteDevicesResourceApi, getDevicesResourcesApi, updateDevicesResourceApi } from '@/containers/Devices/rest'
import { defaultNewResource, resourceModalTypes } from '@/containers/Devices/constants'
import { handleCreateResourceErrors, handleDeleteResourceErrors, handleUpdateResourceErrors } from '@/containers/Devices/utils'
import { DevicesDetailsResourceModalData } from '@/containers/Devices/Detail/DevicesDetailsPage/DevicesDetailsPage.types'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import { history } from '@/store'
import { isNotificationActive, toggleActiveNotification } from '@/containers/Devices/slice'
import { deviceResourceUpdateListener } from '@/containers/Devices/websockets'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { createResourceNotificationId } from '@/containers/PendingCommands/utils'

const Tab2: FC<Props> = (props) => {
    const { deviceStatus, deviceName, isOnline, isActiveTab, isUnregistered, loading, resourcesData, loadingResources, refreshResources } = props
    const {
        id,
        href: hrefParam,
    }: {
        id: string
        href: string
    } = useParams()
    const [resourceModalData, setResourceModalData] = useState<DevicesDetailsResourceModalData | undefined>(undefined)
    const [loadingResource, setLoadingResource] = useState(false)
    const [savingResource, setSavingResource] = useState(false)
    const [deleteResourceHref, setDeleteResourceHref] = useState<string>('')
    const [resourceModal, setResourceModal] = useState(false)
    const [ttlHasError, setTtlHasError] = useState(false)
    const resources = resourcesData?.[0]?.resources || []
    const { formatMessage: _ } = useIntl()
    const isMounted = useIsMounted()
    const wellKnownConfig = security.getWellKnowConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }
    const [ttl, setTtl] = useState(wellKnownConfig?.defaultCommandTimeToLive || 0)

    const { ref, width, height } = useResizeDetector()

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
                    type: resourceModalTypes.CREATE_RESOURCE,
                })
                setResourceModal(true)
            }
        } catch (error) {
            if (error && isMounted.current) {
                setLoadingResource(false)
                Notification.error({
                    title: _(t.resourceRetrieveError),
                    message: getApiErrorMessage(error),
                })
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
                })
                setResourceModal(true)
            }
        } catch (error) {
            if (error && isMounted.current) {
                setLoadingResource(false)
                Notification.error({
                    title: _(t.resourceRetrieveError),
                    message: getApiErrorMessage(error),
                })
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
                    { toastId: createResourceNotificationId(id, href, auditContext?.correlationId, auditContext?.userId) }
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
                // console.log('update done, show toast!')
                Notification.success(
                    { title: _(t.resourceUpdateSuccess), message: _(t.resourceWasUpdated) },
                    {
                        toastId: createResourceNotificationId(id, href, auditContext?.correlationId, auditContext?.userId),
                    }
                )

                handleCloseUpdateModal()
                setSavingResource(false)
            }
        } catch (error) {
            if (error && isMounted.current) {
                handleUpdateResourceErrors(error, { id, href }, _)
                setSavingResource(false)
                handleCloseUpdateModal()
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

        if (hrefParam) {
            // Remove the href from the URL when the update modal is closed
            history.replace(window.location.pathname.replace(`/${hrefParam}`, ''))
        }
    }

    const closeDeleteModal = () => {
        setDeleteResourceHref('')
    }

    return (
        <div
            ref={ref}
            style={{
                height: '100%',
            }}
        >
            <DevicesResources
                data={resources}
                deviceStatus={deviceStatus}
                isActiveTab={isActiveTab}
                loading={loadingResource}
                onCreate={openCreateModal}
                onDelete={openDeleteModal}
                onUpdate={openUpdateModal}
                pageSize={{ width, height: height ? height - 32 : 0 }} // tree switch
            />
            <DevicesResourcesModal
                {...resourceModalData}
                confirmDisabled={ttlHasError}
                createResource={createResource}
                deviceId={id}
                deviceName={deviceName}
                deviceResourceUpdateListener={deviceResourceUpdateListener}
                fetchResource={openUpdateModal}
                i18n={{
                    close: _(t.close),
                    commandTimeout: _(t.commandTimeout),
                    create: _(t.create),
                    creating: _(t.creating),
                    deviceId: _(t.deviceId),
                    interfaces: _(t.interfaces),
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
                onClose={() => setResourceModal(false)}
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
                        ttlHasError={ttlHasError}
                    />
                }
                updateResource={updateResource}
            />
            <DeleteModal
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
        </div>
    )
}

Tab2.displayName = 'Tab2'
export default Tab2
