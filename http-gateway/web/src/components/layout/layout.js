import PropTypes from 'prop-types'
import { Helmet } from 'react-helmet'
import dropRight from 'lodash/dropRight'
import last from 'lodash/last'
import classNames from 'classnames'

import { Breadcrumbs, breadcrumbsShape } from '@/components/breadcrumbs'
import { PageLoader } from '@/components/page-loader'

import { layoutTypes } from './constants'

const { FULL_PAGE, SPLIT } = layoutTypes

/**
 * Basic layout component.
 * @param {Element} header - Elements to be rendered inline with the breadcrumbs, but justified to the end.
 * @param {Array} breadcrumbs - Breadcrumbs to be rendered.
 * @param {Boolean} loading - Display's a loader below status bar.
 * @param {String} title - Sets the title of the browser tab.
 * @param {String} type - Layout type. When set to SPLIT,
 * the first n children will be rendered on the left side and the last child will be rendered to the right side.
 */
export const Layout = props => {
  const { header, breadcrumbs, loading, title, type, children } = props
  const isSplit = type === SPLIT && Array.isArray(children)

  return (
    <>
      <Helmet>
        <title>{title}</title>
      </Helmet>
      <PageLoader loading={loading} />
      <div id="layout">
        {(breadcrumbs || header) && (
          <div className="layout-header">
            {breadcrumbs && <Breadcrumbs items={breadcrumbs} />}
            <div>{header}</div>
          </div>
        )}
        <div className={classNames('layout-content', { split: isSplit })}>
          {isSplit ? (
            <>
              <div className="layout-split-common layout-left">
                {dropRight(children)}
              </div>
              {children.length > 1 && (
                <div className="layout-split-common layout-right">
                  {last(children)}
                </div>
              )}
            </>
          ) : (
            children
          )}
        </div>
      </div>
    </>
  )
}

Layout.propTypes = {
  breadcrumbs: breadcrumbsShape,
  loading: PropTypes.bool,
  header: PropTypes.node,
  title: PropTypes.string,
  type: PropTypes.oneOf([FULL_PAGE, SPLIT]),
}

Layout.defaultProps = {
  breadcrumbs: null,
  loading: false,
  header: null,
  title: null,
  type: FULL_PAGE,
}
