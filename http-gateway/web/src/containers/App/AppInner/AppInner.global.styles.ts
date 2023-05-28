import { css } from '@emotion/react'

export const globalStyle = (toastNotifications = false) =>
    toastNotifications
        ? css``
        : css`
              .notification-toast {
                  opacity: 0 !important;
                  visibility: hidden !important;
                  width: 0 !important;
                  height: 0 !important;
                  overflow: hidden !important;
                  min-height: 0 !important;
                  padding: 0 !important;
                  margin: 0 !important;
                  border: 0 !important;
                  box-shadow: none !important;
              }
          `
