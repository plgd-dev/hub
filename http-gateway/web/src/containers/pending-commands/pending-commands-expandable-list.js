import { useEffect, useState } from 'react'
import ReactDOM from 'react-dom'
import { useIntl } from 'react-intl'
import classNames from 'classnames'
import PropTypes from 'prop-types'

import { PendingCommandsList } from './_pending-commands-list'
import { messages as t } from './pending-commands-i18n'

export const PendingCommandsExpandableList = ({ deviceId }) => {
  const [domReady, setDomReady] = useState(false)
  const [expanded, setExpanded] = useState(false)
  const { formatMessage: _ } = useIntl()

  useEffect(() => {
    setDomReady(true)
  }, [])

  const toggleExpanded = () => {
    setExpanded(prev => !prev)
  }

  return (
    <>
      {domReady &&
        ReactDOM.createPortal(
          <span className="link expander-button" onClick={toggleExpanded}>
            {_(t.recentTasks)}
            <i
              className={classNames('fas', {
                'fa-chevron-down': expanded,
                'fa-chevron-up': !expanded,
              })}
            />
          </span>,
          document.querySelector('#footer .left')
        )}
      {expanded && (
        <div id="pending-commands-expandable-box">
          <PendingCommandsList embedded deviceId={deviceId} />
        </div>
      )}
    </>
  )
}

PendingCommandsExpandableList.propTypes = {
  deviceId: PropTypes.string,
}

PendingCommandsExpandableList.defaultProps = {
  deviceId: null,
}
