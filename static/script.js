const app = Vue.createApp({
    data() {
        // get dataset ID from query string
        const datasetID = new URLSearchParams(window.location.search).get('datasetID');
        return {
            comments: [],
            saveTimeout: null,
            heightAdjustTimeout: null,
            filterText: "",
            categoryFilter: "",
            sortKey: 'category',
            colours: [
                "#fef2f2", // red
                "#fdf4ff", // purple
                "#f5f3ff", // violet
                "#eff6ff", // blue
                "#f0fdf4", // green
                "#fefce8", // yellow
                "#fff7ed", // orange
            ],
            datasetID: datasetID,
            focusedCommentID: null,
        }
    },


    computed: {
        filteredComments() {
            if (!this.filterText && !this.categoryFilter) return this.comments
            return this.comments.filter(comment => {
                const matchesText = !this.filterText ||
                    comment.comment.toLowerCase().includes(this.filterText.toLowerCase())
                const matchesCategory = !this.categoryFilter ||
                    comment.category.toLowerCase().includes(this.categoryFilter.toLowerCase())
                return (matchesText && matchesCategory) || this.focusedCommentID === comment.id
            })
        },
        categoryStats() {
            const stats = {}
            this.comments.forEach(comment => {
                stats[comment.category] = (stats[comment.category] || 0) + 1
            })
            const statsPairs = Object.entries(stats)
                .map(([category, count]) => ({ category, count }))
            if (this.sortKey === 'count') {
                return statsPairs.sort((a, b) => b.count - a.count)
            } else {
                return statsPairs.sort((a, b) => a.category.localeCompare(b.category))
            }

        },
        uniqueCategories() {
            const categories = new Set(this.comments.map(c => c.category))
            return [...categories].sort()
        },
    },

    methods: {
        async fetchComments() {
            const response = await fetch('/api/comments/' + this.datasetID)
            this.comments = (await response.json()).map(c => ({
                ...c,
                showSuggestions: false,
                edited: false
            }))
        },

        async saveComment(comment) {
            this.focusedCommentID = null;
            try {
                await fetch(`/api/comments/${comment.id}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(comment)
                })
                comment.edited = false
            } catch (err) {
                console.error('Error saving:', err)
            }
        },

        getCategoryColor(category) {
            const hash = category.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
            const length = this.colours.length;
            return this.colours[hash % length];
        },

        adjustTextareaHeight() {
            const textarea = this.$refs.textareaRef[0];
            textarea.style.height = 'auto';
            textarea.style.height = textarea.scrollHeight + 'px';
            textarea.focus();
        },
    },
    async mounted() {
        await this.fetchComments()
    },

})

app.mount('#app')
