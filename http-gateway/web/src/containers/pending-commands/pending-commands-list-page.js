import { useEffect } from 'react'
import { useIntl } from 'react-intl'
import { toast } from 'react-toastify'

import { WebSocketEventClient, eventFilters } from '@/common/services'
import { Layout } from '@/components/layout'
import { getApiErrorMessage } from '@/common/utils'
import { messages as menuT } from '@/components/menu/menu-i18n'

import { PendingCommandsList } from './_pending-commands-list'
import { usePendingCommandsList } from './hooks'
import {
  NEW_PENDING_COMMAND_WS_KEY,
  UPDATE_PENDING_COMMANDS_WS_KEY,
} from './constants'
import {
  handleEmitNewPendingCommand,
  handleEmitUpdatedCommandEvents,
} from './utils'
// import { messages as t } from './pending-commands-i18n'

export const PendingCommandsListPage = () => {
  const { formatMessage: _ } = useIntl()
  const { data, loading, error } = usePendingCommandsList()

  useEffect(() => {
    if (error) {
      toast.error(getApiErrorMessage(error))
    }
  }, [error])

  useEffect(() => {
    // WS for adding a new pending command to the list
    WebSocketEventClient.subscribe(
      {
        eventFilter: [
          eventFilters.RESOURCE_CREATE_PENDING,
          eventFilters.RESOURCE_DELETE_PENDING,
          eventFilters.RESOURCE_UPDATE_PENDING,
          eventFilters.RESOURCE_RETRIEVE_PENDING,
          eventFilters.DEVICE_METADATA_UPDATE_PENDING,
        ],
      },
      NEW_PENDING_COMMAND_WS_KEY,
      handleEmitNewPendingCommand
    )

    // WS for updating the status of a pending command
    WebSocketEventClient.subscribe(
      {
        eventFilter: [
          eventFilters.RESOURCE_CREATED,
          eventFilters.RESOURCE_DELETED,
          eventFilters.RESOURCE_UPDATED,
          eventFilters.RESOURCE_RETRIEVED,
          eventFilters.DEVICE_METADATA_UPDATED,
        ],
      },
      UPDATE_PENDING_COMMANDS_WS_KEY,
      handleEmitUpdatedCommandEvents
    )

    return () => {
      WebSocketEventClient.unsubscribe(NEW_PENDING_COMMAND_WS_KEY)
      WebSocketEventClient.unsubscribe(UPDATE_PENDING_COMMANDS_WS_KEY)
    }
  }, [])

  const onCancelClick = () => {}

  const onViewClick = () => {}

  return (
    <Layout
      title={_(menuT.pendingCommands)}
      breadcrumbs={[
        {
          to: '/',
          label: _(menuT.dashboard),
        },
        {
          label: _(menuT.pendingCommands),
        },
      ]}
      loading={loading}
    >
      <PendingCommandsList
        data={data}
        onCancelClick={onCancelClick}
        onViewClick={onViewClick}
      />
    </Layout>
  )
}
