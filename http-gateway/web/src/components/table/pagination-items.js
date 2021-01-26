import PropTypes from 'prop-types'
import BPagination from 'react-bootstrap/Pagination'

export const PaginationItems = ({
  activePage,
  pageCount,
  maxButtons,
  boundaryLinks,
  ellipsis,
  onItemClick,
}) => {
  const pageButtons = []

  let startPage
  let endPage

  if (maxButtons && maxButtons < pageCount) {
    startPage = Math.max(
      Math.min(
        activePage - Math.floor(maxButtons / 2, 10),
        pageCount - maxButtons + 1
      ),
      1
    )
    endPage = startPage + maxButtons - 1
  } else {
    startPage = 1
    endPage = pageCount
  }

  for (let page = startPage; page <= endPage; ++page) {
    pageButtons.push(
      <BPagination.Item
        onClick={() => onItemClick(page - 1)}
        key={page}
        active={page === activePage}
      >
        {page}
      </BPagination.Item>
    )
  }

  if (ellipsis && boundaryLinks && startPage > 1) {
    if (startPage > 2) {
      pageButtons.unshift(<BPagination.Ellipsis key="ellipsisFirst" disabled />)
    }

    pageButtons.unshift(
      <BPagination.Item onClick={() => onItemClick(1)} key={1} active={false}>
        {'1'}
      </BPagination.Item>
    )
  }

  if (ellipsis && endPage < pageCount) {
    if (!boundaryLinks || endPage < pageCount - 1) {
      pageButtons.push(<BPagination.Ellipsis key="ellipsis" disabled />)
    }

    if (boundaryLinks) {
      pageButtons.push(
        <BPagination.Item
          onClick={() => onItemClick(pageCount - 1)}
          key={pageCount}
          active={false}
        >
          {pageCount}
        </BPagination.Item>
      )
    }
  }

  return pageButtons
}

PaginationItems.propTypes = {
  activePage: PropTypes.number.isRequired,
  pageCount: PropTypes.number.isRequired,
  onItemClick: PropTypes.func.isRequired,
  maxButtons: PropTypes.number,
  boundaryLinks: PropTypes.bool,
  ellipsis: PropTypes.bool,
}

PaginationItems.defaultProps = {
  maxButtons: 0,
  boundaryLinks: true,
  ellipsis: true,
}
