import { createSlice } from '@reduxjs/toolkit'
import { remoteClientStatuses } from '@/containers/RemoteClients/contacts'

export type RemoteClientStatusType = (typeof remoteClientStatuses)[keyof typeof remoteClientStatuses]

export type RemoteClientType = {
    id: string
    clientName: string
    clientIP: string
    created: string
    status: RemoteClientStatusType
    version: string
}

export type StoreType = {
    remoteClients: RemoteClientType[]
}

const initialState: StoreType = {
    remoteClients: [
        {
            id: 'FAj9X1Gxs9rLm-62r6yGJ',
            created: '2023-07-22T17:58:11.427Z',
            version: '0.6.0',
            clientName: 'Test',
            clientIP: 'http://212.89.237.161:50080',
            status: remoteClientStatuses.REACHABLE,
        },
        {
            id: 'ascaa-62r6yGJ',
            created: '2023-07-22T17:58:11.427Z',
            version: '0.6.0',
            clientName: 'Test no working',
            clientIP: 'http://localhost:50080',
            status: remoteClientStatuses.REACHABLE,
        },
        {
            id: 'ascaa-62r6yGJpop',
            created: '2023-07-22T17:58:11.427Z',
            version: '0.6.0',
            clientName: 'PM local',
            clientIP: 'http://localhost:3001',
            status: remoteClientStatuses.REACHABLE,
        },
    ],
}

const { reducer, actions } = createSlice({
    name: 'remoteClients',
    initialState,
    reducers: {
        addRemoteClient(state, { payload }) {
            state.remoteClients.push(payload)
        },
        deleteRemoteClients(state, { payload }) {
            payload.forEach((remoteClientId: string) => {
                state.remoteClients.splice(
                    state.remoteClients.findIndex((remoteClient) => remoteClient.id === remoteClientId),
                    1
                )
            })
        },
        updateRemoteClients(state, { payload }) {
            payload.forEach((remoteClient: RemoteClientType) => {
                const index = state.remoteClients.findIndex((originRemoteClient) => originRemoteClient.id === remoteClient.id)

                if (index >= 0) {
                    state.remoteClients[index] = remoteClient
                }
            })
        },
    },
})

// Actions
export const { addRemoteClient, deleteRemoteClients, updateRemoteClients } = actions

// Reducer
export default reducer