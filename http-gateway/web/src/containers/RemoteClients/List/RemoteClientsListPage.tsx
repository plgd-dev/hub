import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { nanoid } from '@reduxjs/toolkit'
import { useDispatch, useSelector } from 'react-redux'
import { Link, useNavigate } from 'react-router-dom'

import { fetchApi } from '@shared-ui/common/services'
import { IconTrash } from '@shared-ui/components/Atomic/Icon'
import DeleteModal from '@shared-ui/components/Atomic/Modal/components/DeleteModal'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { RemoteClientType } from '@shared-ui/app/clientApp/RemoteClients/RemoteClients.types'
import { remoteClientStatuses, RemoteClientStatusesType } from '@shared-ui/app/clientApp/RemoteClients/constants'
import { hasDifferentOwner } from '@shared-ui/common/services/api-utils'
import StatusPill from '@shared-ui/components/Atomic/StatusPill'
import TableActionButton from '@shared-ui/components/Organisms/TableActionButton'
import IconEdit from '@shared-ui/components/Atomic/Icon/components/IconEdit'
import { states } from '@shared-ui/components/Atomic/StatusPill/constants'

import { messages as t } from '../RemoteClients.i18n'
import { messages as g } from '../../Global.i18n'
import RemoteClientsListHeader from './RemoteClientsListHeader'
import AddRemoteClientModal from '@/containers/RemoteClients/List/AddRemoteClientModal/AddRemoteClientModal'
import { ClientInformationLineType } from '@/containers/RemoteClients/List/AddRemoteClientModal/AddRemoteClientModal.types'
import { addRemoteClient, deleteRemoteClients, updateRemoteClients, updateRemoteClient } from '@/containers/RemoteClients/slice'
import { CombinedStoreType } from '@/store/store'
import notificationId from '@/notificationId'
import PageLayout from '@/containers/Common/PageLayout'
import { NO_DEVICE_NAME } from '@/containers/Devices/constants'
import TableList from '@/containers/Common/TableList/TableList'

const RemoteClientsListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const [addClientModal, setAddClientModal] = useState(false)
    const [dataLoading, setDataLoading] = useState(false)
    const [remoteClients, setRemoteClients] = useState<RemoteClientType[] | undefined>(undefined)
    const [selected, setSelected] = useState<string[]>([])
    const [unselectRowsToken, setUnselectRowsToken] = useState(1)

    const navigate = useNavigate()

    const [editRemoteClientId, setEditRemoteClientId] = useState<undefined | string>(undefined)

    const dispatch = useDispatch()
    const storedRemoteStore = useSelector((state: CombinedStoreType) => state.remoteClients)

    const selectedCount = useMemo(() => selected.length, [selected])
    const selectedRemoteClient = useMemo(
        () => (selectedCount === 1 && remoteClients ? remoteClients.find?.((remoteClient) => remoteClient.id === selected[0]) : null),
        [remoteClients, selected, selectedCount]
    )

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

    const handleOpenDeleteModal = useCallback((_isAllSelected: boolean, selection: string[]) => {
        setSelected(selection)
    }, [])

    const handleCloseDeleteModal = useCallback(() => {
        setSelected([])
        setUnselectRowsToken((prev) => prev + 1)
    }, [])

    const deleteClients = useCallback(() => {
        dispatch(deleteRemoteClients(selected))

        Notification.success(
            { title: _(t.clientsDeleted), message: _(t.clientsDeletedMessage) },
            { notificationId: notificationId.HUB_REMOTE_CLIENTS_LIST_PAGE_DELETE_CLIENTS }
        )

        setSelected([])
        setUnselectRowsToken((prevValue) => prevValue + 1)

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [selected])

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

                        if (hasDifferentOwner(value.data, remoteClient)) {
                            if (remoteClient.status !== remoteClientStatuses.DIFFERENT_OWNER) {
                                dataForUpdate.push({ ...remoteClient, status: remoteClientStatuses.DIFFERENT_OWNER })
                            }

                            return {
                                ...remoteClient,
                                status: remoteClientStatuses.DIFFERENT_OWNER,
                            }
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

        setDataLoading(false)

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [storedRemoteStore.remoteClients])

    const getStatusData = useCallback((status: RemoteClientStatusesType) => {
        switch (status) {
            case remoteClientStatuses.DIFFERENT_OWNER:
                return {
                    message: _(t.occupied),
                    status: states.OCCUPIED,
                }
            case remoteClientStatuses.UNREACHABLE:
                return {
                    message: _(t.unReachable),
                    status: states.OFFLINE,
                }
            case remoteClientStatuses.REACHABLE:
            default:
                return {
                    message: _(t.reachable),
                    status: states.ONLINE,
                }
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const columns = useMemo(
        () => [
            {
                Header: _(g.name),
                accessor: 'clientName',
                Cell: ({ value, row }: { value: string; row: any }) => {
                    const remoteClientName = value || NO_DEVICE_NAME

                    if (row.original.status === remoteClientStatuses.UNREACHABLE) {
                        return <span>{remoteClientName}</span>
                    }
                    return (
                        <Link to={`/remote-clients/${row.original?.id}`}>
                            <span className='no-wrap-text'>{remoteClientName}</span>
                        </Link>
                    )
                },
            },
            {
                Header: _(t.ipAddress),
                accessor: 'clientUrl',
                style: { width: '350px' },
                Cell: ({ value }: { value: string }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(g.status),
                accessor: 'status',
                style: { width: '200px' },
                Cell: ({ row }: { row: any }) => {
                    const statusData = getStatusData(row.original.status)
                    return <StatusPill label={statusData.message} status={statusData.status} />
                },
            },
            {
                Header: _(t.version),
                accessor: 'version',
                style: { width: '200px' },
                Cell: ({ value }: { value: string }) => <span className='no-wrap-text'>{value}</span>,
            },
            {
                Header: _(g.action),
                accessor: 'action',
                style: { width: '66px' },
                disableSortBy: true,
                Cell: ({ row }: any) => (
                    <TableActionButton
                        items={[
                            {
                                onClick: () => handleOpenDeleteModal(false, [row.original.id]),
                                label: _(g.delete),
                                icon: <IconTrash />,
                            },
                            {
                                onClick: () => navigate(`/remote-clients/${row.original.id}/configuration`),
                                label: _(g.edit),
                                icon: <IconEdit />,
                            },
                        ]}
                    />
                ),
                className: 'actions',
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <PageLayout
            breadcrumbs={[{ label: _(t.remoteClients), link: '/' }]}
            header={<RemoteClientsListHeader dataLoading={dataLoading} onClientClick={() => setAddClientModal(true)} />}
            title={_(t.remoteUiClient)}
        >
            <AddRemoteClientModal
                closeOnBackdrop={false}
                defaultAuthMode={editRemoteClientData?.deviceAuthenticationMode}
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
                                  attributeKey: 'deviceAuthenticationMode',
                                  value: editRemoteClientData?.deviceAuthenticationMode,
                              },
                          ]
                        : undefined
                }
                defaultClientName={editRemoteClientData?.clientName}
                defaultClientUrl={editRemoteClientData?.clientUrl}
                defaultPreSharedKey={editRemoteClientData?.preSharedKey}
                defaultPreSharedSubjectId={editRemoteClientData?.preSharedSubjectId}
                onClose={() => {
                    setAddClientModal(false)
                    setEditRemoteClientId(undefined)
                }}
                onFormSubmit={handleClientAdd}
                show={addClientModal}
            />

            <TableList
                columns={columns}
                data={remoteClients}
                defaultSortBy={[
                    {
                        id: 'clientName',
                        desc: false,
                    },
                ]}
                i18n={{
                    multiSelected: _(t.remoteClients),
                    singleSelected: _(t.remoteClient),
                }}
                onDeleteClick={handleOpenDeleteModal}
                primaryAttribute='clientName'
                unselectRowsToken={unselectRowsToken}
            />

            <DeleteModal
                fullSizeButtons
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
                maxWidth={440}
                maxWidthTitle={320}
                minWidth={440}
                onClose={handleCloseDeleteModal}
                show={selectedCount > 0}
                subTitle={selectedCount === 1 && selectedRemoteClient ? selectedRemoteClient?.clientName : undefined}
                title={selectedCount === 1 ? _(t.deleteClientMessage) : _(t.deleteClientsMessage, { count: selectedCount })}
            />
        </PageLayout>
    )
}

RemoteClientsListPage.displayName = 'RemoteClientsListPage'

export default RemoteClientsListPage
