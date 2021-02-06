import { ToastContainer as Toastr } from 'react-toastify'

export const ToastContainer = () => {
  return (
    <Toastr
      closeButton={({ closeToast }) => (
        <i onClick={closeToast} className="fas fa-times" />
      )}
    />
  )
}
