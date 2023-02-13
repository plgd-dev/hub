import { FC, useEffect, useState } from 'react'
import ReactDOM from 'react-dom'
import { useIntl } from 'react-intl'
import classNames from 'classnames'
import { Props } from './PendingCommandsExpandableList.types'
import PendingCommandsList from '../PendingCommandsList'
import { messages as t } from '../PendingCommands.i18n'

const PendingCommandsExpandableList: FC<Props> = ({ deviceId }) => {
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
          document.querySelector('#footer #pending-commands-wrapper') as Element
        )}
      {expanded && (
        <div id="pending-commands-expandable-box">
          <PendingCommandsList embedded deviceId={deviceId} />
        </div>
      )}
    </>
  )
}

PendingCommandsExpandableList.displayName = 'PendingCommandsExpandableList'

export default PendingCommandsExpandableList
