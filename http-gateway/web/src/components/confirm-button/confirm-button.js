import { useState } from 'react'
import PropTypes from 'prop-types'

import { ConfirmModal } from '@/components/confirm-modal'
import { Button } from '@/components/button'

const NOOP = () => {}

export const ConfirmButton = ({
  onConfirm,
  children,
  confirmButtonText,
  cancelButtonText,
  title,
  body,
  loading,
  closeOnConfirm,
  modalProps,
  ...rest
}) => {
  const [show, setShow] = useState(false)

  const open = () => {
    setShow(true)
  }

  const close = () => {
    setShow(false)
  }

  const onConfirmClick = onClose => {
    onConfirm(onClose)

    if (closeOnConfirm) {
      setShow(false)
    }
  }

  return (
    <>
      <Button {...rest} loading={loading} onClick={open}>
        {children}
      </Button>
      <ConfirmModal
        {...modalProps}
        show={show}
        onClose={!loading ? close : NOOP}
        title={title}
        body={body}
        loading={loading}
        cancelButtonText={cancelButtonText}
        confirmButtonText={confirmButtonText}
        onConfirm={onConfirmClick}
      />
    </>
  )
}

ConfirmButton.propTypes = {
  onConfirm: PropTypes.func.isRequired,
  title: PropTypes.node.isRequired,
  body: PropTypes.node.isRequired,
  confirmButtonText: PropTypes.string,
  cancelButtonText: PropTypes.string,
  closeOnConfirm: PropTypes.bool,
  loading: PropTypes.bool,
}

ConfirmButton.defaultProps = {
  confirmButtonText: null,
  cancelButtonText: null,
  closeOnConfirm: true,
  loading: false,
}
