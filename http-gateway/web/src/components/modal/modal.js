import PropTypes from 'prop-types'
import BModal from 'react-bootstrap/Modal'
import classNames from 'classnames'

import { useAppConfig } from '@/containers/App'

export const Modal = ({
  onClose,
  title,
  renderBody,
  renderFooter,
  backdropClassName,
  dialogClassName,
  show,
  closeButton,
  ...rest
}) => {
  const { collapsed } = useAppConfig()

  return (
    <BModal
      show={show}
      onHide={onClose}
      centered
      backdropClassName={classNames({ collapsed }, backdropClassName)}
      dialogClassName={classNames({ collapsed }, dialogClassName)}
      {...rest}
    >
      {show && (
        <>
          <BModal.Header>
            <BModal.Title>{title || ''}</BModal.Title>
            {closeButton && (
              <button
                className="close"
                aria-hidden="true"
                aria-label="Close"
                onClick={onClose}
              >
                <span className="fas fa-times" />
              </button>
            )}
          </BModal.Header>
          <BModal.Body>{renderBody && renderBody()}</BModal.Body>
          <BModal.Footer>{renderFooter && renderFooter()}</BModal.Footer>
        </>
      )}
    </BModal>
  )
}

Modal.propTypes = {
  show: PropTypes.bool.isRequired,
  title: PropTypes.node,
  renderBody: PropTypes.func,
  renderFooter: PropTypes.func,
  closeButton: PropTypes.bool,
  onClose: PropTypes.func,
}

Modal.defaultProps = {
  closeButton: true,
  onClose: () => {},
}
