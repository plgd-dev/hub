// import { useEffect, useState, useRef } from 'react'
import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'

// import { Editor } from '@/components/editor'
// import { Select } from '@/components/select'
// import { Button } from '@/components/button'
// import { Badge } from '@/components/badge'
// import { Label } from '@/components/label'
import { Modal } from '@/components/modal'

import { commandTypes } from './constants'
import { messages as t } from './pending-commands-i18n'

export const PendingCommandDetailsModal = ({
  commandType,
  onClose,
  content,
}) => {
  const { formatMessage: _ } = useIntl()

  const renderBody = () => {
    return (
      <div className="json-object-box">
        <pre>{JSON.stringify(content, null, 2)}</pre>
      </div>
    )
  }

  const title = commandType ? `${_(t[commandType])} ${_(t.commandContent)}` : null

  return (
    <Modal
      show={!!commandType}
      onClose={onClose}
      title={title}
      renderBody={renderBody}
    />
  )
}

PendingCommandDetailsModal.propTypes = {
  onClose: PropTypes.func.isRequired,
  content: PropTypes.object,
  commandType: PropTypes.oneOf(Object.values(commandTypes)),
}

PendingCommandDetailsModal.defaultProps = {
  commandType: null,
  content: {},
}
