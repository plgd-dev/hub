import React, { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import isFunction from 'lodash/isFunction'
import { nanoid } from '@reduxjs/toolkit'
import { useDispatch, useSelector } from 'react-redux'

import PageLayout from '@shared-ui/components/Atomic/PageLayout'
import Footer from '@shared-ui/components/Layout/Footer'
import { fetchApi } from '@shared-ui/common/services'
import { DeleteModal } from '@shared-ui/components/Atomic'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { RemoteClientType } from '@shared-ui/app/clientApp/RemoteClients/RemoteClients.types'
import { remoteClientStatuses } from '@shared-ui/app/clientApp/RemoteClients/constants'

import { messages as t } from '../RemoteClients.i18n'
import { messages as g } from '../../Global.i18n'
import { AppContext } from '@/containers/App/AppContext'
import RemoteClientsListHeader from './RemoteClientsListHeader'
import AddRemoteClientModal from '@/containers/RemoteClients/List/AddRemoteClientModal/AddRemoteClientModal'
import { ClientInformationLineType } from '@/containers/RemoteClients/List/AddRemoteClientModal/AddRemoteClientModal.types'
import { addRemoteClient, deleteRemoteClients, updateRemoteClients, updateRemoteClient } from '@/containers/RemoteClients/slice'
import { CombinedStoreType } from '@/store/store'
import RemoteClientsList from '@/containers/RemoteClients/List/RemoteClientsList'
import notificationId from '@/notificationId'

const RemoteClientsListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const { footerExpanded, setFooterExpanded } = useContext(AppContext)
    const [addClientModal, setAddClientModal] = useState(false)
    const [dataLoading, setDataLoading] = useState(false)
    const [remoteClients, setRemoteClients] = useState<RemoteClientType[] | undefined>(undefined)
    const [isAllSelected, setIsAllSelected] = useState(false)
    const [deleteModalOpen, setDeleteModalOpen] = useState(false)
    const [singleDelete, setSingleDelete] = useState<null | string>(null)
    const [selectedClients, setSelectedClients] = useState([])
    const [unselectRowsToken, setUnselectRowsToken] = useState(1)

    const [editRemoteClientId, setEditRemoteClientId] = useState<undefined | string>(undefined)

    const dispatch = useDispatch()
    const storedRemoteStore = useSelector((state: CombinedStoreType) => state.remoteClients)

    const combinedSelectedClients = useMemo(() => (singleDelete ? [singleDelete] : selectedClients), [singleDelete, selectedClients])

    const selectedClientsCount = combinedSelectedClients.length
    const selectedRemoteClient =
        selectedClientsCount === 1 && remoteClients ? remoteClients.find?.((remoteClient) => remoteClient.id === combinedSelectedClients[0]) : null

    const editRemoteClientData = useMemo(() => {
        const remoteClientData = remoteClients?.find((remoteClient) => remoteClient.id === editRemoteClientId)
        if (remoteClientData) {
            setAddClientModal(true)

            return remoteClientData
        }

        return undefined
    }, [editRemoteClientId, remoteClients])

    const handleClientAdd = useCallback(
        (clientInformation: ClientInformationLineType[]) => {
            setAddClientModal(false)

            const dataForSave: { [key: string]: string } = {}
            clientInformation.forEach((client) => (dataForSave[client.attributeKey] = client.value))

            if (editRemoteClientData) {
                dispatch(updateRemoteClient({ ...editRemoteClientData, ...dataForSave }))
                setEditRemoteClientId(undefined)

                Notification.success(
                    { title: _(t.clientsUpdated), message: _(t.clientsUpdatedMessage) },
                    { notificationId: notificationId.HUB_REMOTE_CLIENTS_UPDATE_REMOTE_CLIENT }
                )
            } else {
                dispatch(
                    addRemoteClient({
                        id: nanoid(),
                        created: new Date(),
                        status: remoteClientStatuses.REACHABLE,
                        ...dataForSave,
                    })
                )
            }
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [editRemoteClientData]
    )

    const handleOpenDeleteModal = useCallback(
        (clientId?: string) => {
            if (typeof clientId === 'string') {
                setSingleDelete(clientId)
            } else if (singleDelete && !clientId) {
                setSingleDelete(null)
            }

            setDeleteModalOpen(true)
        },
        [singleDelete]
    )

    const handleCloseDeleteModal = useCallback(() => {
        setSingleDelete(null)
        setDeleteModalOpen(false)
    }, [])

    const deleteClients = useCallback(() => {
        dispatch(deleteRemoteClients(combinedSelectedClients))

        Notification.success(
            { title: _(t.clientsDeleted), message: _(t.clientsDeletedMessage) },
            { notificationId: notificationId.HUB_REMOTE_CLIENTS_LIST_PAGE_DELETE_CLIENTS }
        )

        setSingleDelete(null)
        setDeleteModalOpen(false)
        setUnselectRowsToken((prevValue) => prevValue + 1)

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [combinedSelectedClients])

    useEffect(() => {
        setDataLoading(true)
        const dataForUpdate: RemoteClientType[] = []
        const viewData = storedRemoteStore.remoteClients.map((remoteClient: RemoteClientType) =>
            fetchApi(`${remoteClient.clientUrl}/.well-known/configuration`, {
                useToken: false,
            }).catch((_e) => {
                if (remoteClient.status === remoteClientStatuses.REACHABLE) {
                    dataForUpdate.push({
                        ...remoteClient,
                        status: remoteClientStatuses.UNREACHABLE,
                    })
                }
                return {
                    ...remoteClient,
                    status: remoteClientStatuses.UNREACHABLE,
                }
            })
        )

        Promise.all(viewData)
            .then((values) =>
                values.map((value, index) => {
                    // response from server
                    if (value.hasOwnProperty('statusText')) {
                        const remoteClient = storedRemoteStore.remoteClients[index]

                        if (remoteClient.version !== value.data?.version || remoteClient.status === remoteClientStatuses.UNREACHABLE) {
                            dataForUpdate.push({ ...remoteClient, version: value.data?.version, status: remoteClientStatuses.REACHABLE })
                        }

                        return {
                            ...remoteClient,
                            version: value.data?.version,
                            status: remoteClientStatuses.REACHABLE,
                        }
                    }

                    // UNREACHABLE - caught error
                    return value
                })
            )
            .then((dataForView) => {
                setDataLoading(false)
                setRemoteClients(dataForView)

                if (dataForUpdate.length) {
                    setTimeout(() => {
                        dispatch(updateRemoteClients(dataForUpdate))
                    }, 200)
                }
            })

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [storedRemoteStore.remoteClients])

    return (
        <PageLayout
            breadcrumbs={[
                {
                    label: _(t.remoteUiClient),
                },
            ]}
            footer={
                <Footer
                    footerExpanded={footerExpanded}
                    paginationComponent={<div id='paginationPortalTarget'></div>}
                    recentTasksPortal={<div id='recentTasksPortalTarget'></div>}
                    recentTasksPortalTitle={
                        <span
                            id='recentTasksPortalTitleTarget'
                            onClick={() => {
                                isFunction(setFooterExpanded) && setFooterExpanded(!footerExpanded)
                            }}
                        >
                            {_(t.recentCommands)}
                        </span>
                    }
                    setFooterExpanded={setFooterExpanded}
                />
            }
            header={<RemoteClientsListHeader dataLoading={dataLoading} onClientClick={() => setAddClientModal(true)} />}
            title={_(t.remoteUiClient)}
        >
            <AddRemoteClientModal
                closeOnBackdrop={false}
                defaultAuthMode={editRemoteClientData?.authenticationMode}
                defaultClientInformation={
                    editRemoteClientData
                        ? [
                              {
                                  attribute: _(t.version),
                                  attributeKey: 'version',
                                  value: editRemoteClientData?.version,
                              },
                              {
                                  attribute: _(t.deviceAuthenticationMode),
                                  attributeKey: 'authenticationMode',
                                  value: editRemoteClientData?.authenticationMode,
                              },
                          ]
                        : undefined
                }
                defaultClientName={editRemoteClientData?.clientName}
                defaultClientUrl={editRemoteClientData?.clientUrl}
                defaultPreSharedKey={editRemoteClientData?.preSharedKey}
                defaultPreSharedSubjectId={editRemoteClientData?.preSharedSubjectId}
                onClose={() => setAddClientModal(false)}
                onFormSubmit={handleClientAdd}
                show={addClientModal}
            />
            <RemoteClientsList
                data={remoteClients || []}
                handleOpenDeleteModal={handleOpenDeleteModal}
                handleOpenEditModal={setEditRemoteClientId}
                isAllSelected={isAllSelected}
                selectedClients={selectedClients}
                setIsAllSelected={setIsAllSelected}
                setSelectedClients={setSelectedClients}
                unselectRowsToken={unselectRowsToken}
            />
            <DeleteModal
                footerActions={[
                    {
                        label: _(g.cancel),
                        onClick: handleCloseDeleteModal,
                        variant: 'tertiary',
                    },
                    {
                        label: _(g.delete),
                        onClick: deleteClients,
                        variant: 'primary',
                    },
                ]}
                fullSizeButtons={selectedClientsCount > 1}
                maxWidth={440}
                maxWidthTitle={320}
                onClose={handleCloseDeleteModal}
                show={deleteModalOpen}
                subTitle={selectedClientsCount === 1 && selectedRemoteClient ? selectedRemoteClient?.clientName : null}
                title={selectedClientsCount === 1 ? _(t.deleteClientMessage) : _(t.deleteClientsMessage, { count: selectedClientsCount })}
            />
        </PageLayout>
    )
}

RemoteClientsListPage.displayName = 'RemoteClientsListPage'

export default RemoteClientsListPage
