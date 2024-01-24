import React, { FC } from 'react'
import { useIntl } from 'react-intl'

import Modal from '@shared-ui/components/Atomic/Modal'
import Editor from '@shared-ui/components/Atomic/Editor'

import { messages as t } from '../PendingCommands.i18n'
import { Props } from './PendingCommandDetailsModal.types'

const PendingCommandDetailsModal: FC<Props> = ({ commandType, onClose, content }) => {
    const { formatMessage: _ } = useIntl()

    const renderBody = () => (
        <div className='json-object-box'>
            <Editor disabled height='300px' json={content} />
        </div>
    )

    // @ts-ignore
    const trans = commandType ? _(t[commandType]) : null
    const title = commandType ? `${trans} ${_(t.commandContent)}` : null

    return <Modal onClose={onClose} renderBody={renderBody} show={!!commandType} title={title} />
}

PendingCommandDetailsModal.displayName = 'PendingCommandDetailsModal'

export default PendingCommandDetailsModal
