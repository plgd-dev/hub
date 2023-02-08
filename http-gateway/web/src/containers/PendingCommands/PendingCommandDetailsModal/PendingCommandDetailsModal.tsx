import { FC } from 'react'
import { useIntl } from 'react-intl'

import Modal from '@shared-ui/components/new/Modal'
import { messages as t } from '../PendingCommands.i18n'
import { Props } from './PendingCommandDetailsModal.types'

const PendingCommandDetailsModal: FC<Props> = ({
  commandType,
  onClose,
  content,
}) => {
  const { formatMessage: _ } = useIntl()

  const renderBody = () => (
    <div className="json-object-box">
      <pre>{JSON.stringify(content, null, 2)}</pre>
    </div>
  )

  // @ts-ignore
  const trans = _(t[commandType])
  const title = commandType ? `${trans} ${_(t.commandContent)}` : null

  return (
    <Modal
      show={!!commandType}
      onClose={onClose}
      title={title}
      renderBody={renderBody}
    />
  )
}

PendingCommandDetailsModal.displayName = 'PendingCommandDetailsModal'

export default PendingCommandDetailsModal
