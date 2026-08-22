// Filters the already-rendered subscription rows without disturbing HTMX sort state.
(function () {
    function applySubscriptionSearch() {
        const input = document.getElementById('subscription-search');
        const list = document.getElementById('subscription-list');
        if (!input || !list) return;

        const query = input.value.trim().toLocaleLowerCase();
        const rows = list.querySelectorAll('tbody tr[data-search-name]');
        let visibleRows = 0;

        rows.forEach((row) => {
            const name = (row.dataset.searchName || '').toLocaleLowerCase();
            const matches = query === '' || name.includes(query);
            row.hidden = !matches;
            if (matches) visibleRows++;
        });

        const emptyState = document.getElementById('subscription-search-empty');
        if (emptyState) {
            const noMatches = query !== '' && rows.length > 0 && visibleRows === 0;
            emptyState.classList.toggle('hidden', !noMatches);
        }
    }

    function initializeSubscriptionSearch() {
        const input = document.getElementById('subscription-search');
        if (!input) return;
        input.addEventListener('input', applySubscriptionSearch);
        applySubscriptionSearch();
    }

    document.addEventListener('DOMContentLoaded', initializeSubscriptionSearch);
    document.addEventListener('htmx:afterSwap', (event) => {
        if (event.detail.target && event.detail.target.id === 'subscription-list') {
            applySubscriptionSearch();
        }
    });
})();
