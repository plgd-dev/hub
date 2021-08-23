import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'

import { Button } from '@/components/button'
import { Modal } from '@/components/modal'
import { messages as t } from './confirm-modal-i18n'

const NOOP = () => {}

export const ConfirmModal = ({
  onConfirm,
  confirmButtonText,
  cancelButtonText,
  title,
  body,
  loading,
  show,
  onClose,
  data,
  confirmDisabled,
  ...rest
}) => {
  const { formatMessage: _ } = useIntl()

  const renderFooter = () => {
    return (
      <div className="w-100 d-flex justify-content-end align-items-center">
        <Button variant="secondary" onClick={onClose} disabled={loading}>
          {cancelButtonText || _(t.cancel)}
        </Button>
        <Button
          variant="primary"
          onClick={() => onConfirm(onClose, data)}
          loading={loading}
          disabled={loading || confirmDisabled}
        >
          {confirmButtonText || _(t.confirm)}
        </Button>
      </div>
    )
  }

  const renderBody = () => body

  return (
    <Modal
      {...rest}
      show={show}
      onClose={!loading ? onClose : NOOP}
      title={title}
      renderBody={renderBody}
      renderFooter={renderFooter}
      closeButton={!loading}
    />
  )
}

ConfirmModal.propTypes = {
  onConfirm: PropTypes.func.isRequired,
  onClose: PropTypes.func.isRequired,
  title: PropTypes.node.isRequired,
  body: PropTypes.node.isRequired,
  show: PropTypes.bool,
  confirmButtonText: PropTypes.string,
  cancelButtonText: PropTypes.string,
  loading: PropTypes.bool,
  confirmDisabled: PropTypes.bool,
  data: PropTypes.object,
}

ConfirmModal.defaultProps = {
  confirmButtonText: null,
  cancelButtonText: null,
  show: false,
  loading: false,
  confirmDisabled: false,
  data: {},
}
