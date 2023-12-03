import { createSlice } from '@reduxjs/toolkit'
import get from 'lodash/get'

import { RemoteClientType } from '@shared-ui/app/clientApp/RemoteClients/RemoteClients.types'
import { remoteClientStatuses } from '@shared-ui/app/clientApp/RemoteClients/constants'

export type StoreType = {
    remoteClients: RemoteClientType[]
}

const isDev = get(process.env, 'REACT_APP_TEST_REMOTE_CLIENTS_MOCK_DATA', false)

const initialState: StoreType = {
    remoteClients: isDev
        ? [
              {
                  id: '123',
                  created: '2023-07-22T17:58:11.427Z',
                  version: '0.6.0',
                  clientName: 'Test PRE_SHARED_KEY',
                  clientUrl: 'https://212.89.237.161:50080',
                  status: remoteClientStatuses.REACHABLE,
                  deviceAuthenticationMode: 'PRE_SHARED_KEY',
                  preSharedSubjectId: 'a',
                  preSharedKey: 'a',
              },
              {
                  id: '456',
                  created: '2023-07-22T17:58:11.427Z',
                  version: '0.6.0',
                  clientName: 'Test X509',
                  clientUrl: 'https://212.89.237.161:50080',
                  status: remoteClientStatuses.REACHABLE,
                  deviceAuthenticationMode: 'X509',
                  preSharedSubjectId: '',
                  preSharedKey: '',
              },
          ]
        : [],
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
        deleteAllRemoteClients(state) {
            state.remoteClients = []
        },
        updateRemoteClient(state, { payload }) {
            const index = state.remoteClients.findIndex((originRemoteClient) => originRemoteClient.id === payload.id)

            if (index >= 0) {
                // const remoteClient = state.remoteClients[index]
                state.remoteClients[index] = {
                    ...state.remoteClients[index],
                    ...payload,
                }
            }
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
export const { addRemoteClient, deleteRemoteClients, deleteAllRemoteClients, updateRemoteClients, updateRemoteClient } = actions

// Reducer
export default reducer
