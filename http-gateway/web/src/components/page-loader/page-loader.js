import { memo } from 'react'
import PropTypes from 'prop-types'

export const PageLoader = memo(({ loading }) => {
  return loading ? (
    <div id="page-loader" role="alert" aria-busy="true">
      <div className="bar">
        <div className="progress">
          <div className="subline inc" />
          <div className="subline dec" />
        </div>
      </div>
    </div>
  ) : null
})

PageLoader.propTypes = {
  loading: PropTypes.bool.isRequired,
}
