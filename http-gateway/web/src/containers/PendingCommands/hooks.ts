import { useContext, useEffect, useState } from 'react'

import { useStreamApi, useEmitter } from '@shared-ui/common/hooks'
import { security } from '@shared-ui/common/services'
import AppContext from '@shared-ui/app/share/AppContext'

import { pendingCommandsApiEndpoints, NEW_PENDING_COMMAND_WS_KEY, UPDATE_PENDING_COMMANDS_WS_KEY, PENDING_COMMANDS_LIST_REFRESH_INTERVAL_MS } from './constants'
import { convertPendingCommandsList, updatePendingCommandsDataStatus } from './utils'
import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'

const getConfig = () => security.getGeneralConfig() as SecurityConfig

export const usePendingCommandsList = (deviceId?: string) => {
    const filter = deviceId ? `?deviceIdFilter=${deviceId}&includeHiddenResources=true` : `?includeHiddenResources=true`
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const { data, updateData, ...rest }: StreamApiPropsType = useStreamApi(
        `${getConfig().httpGatewayAddress}${pendingCommandsApiEndpoints.PENDING_COMMANDS}${filter}`,
        {
            telemetryWebTracer,
            telemetrySpan: 'get-pending-commands',
            env: process.env,
            unauthorizedCallback,
        }
    )

    // Add a new pending command when a WS event is emitted
    useEmitter(NEW_PENDING_COMMAND_WS_KEY, (newCommand: string) => {
        updateData((data || []).concat(newCommand))
    })

    useEmitter(UPDATE_PENDING_COMMANDS_WS_KEY, (updated: { deviceId: string; href: string; correlationId: string; status: string }) => {
        updated.correlationId && updateData(updatePendingCommandsDataStatus(data, updated))
    })

    return {
        ...rest,
        data: data ? convertPendingCommandsList(data) : [],
        updateData,
    }
}

export const useCurrentTime = () => {
    const [currentTime, setCurrentTime] = useState(Date.now())

    useEffect(() => {
        const timeout = setInterval(() => {
            setCurrentTime(Date.now())
        }, PENDING_COMMANDS_LIST_REFRESH_INTERVAL_MS)

        return () => {
            clearInterval(timeout)
        }
    }, [])

    return { currentTime }
}
