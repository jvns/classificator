const app = Vue.createApp({
    data() {
        return {
            comments: [],
            saveTimeout: null,
            filterText: ""
        }
    },

    computed: {
        filteredComments() {
            if (!this.filterText) return this.comments
            const filter = this.filterText.toLowerCase()
            return this.comments.filter(comment =>
                comment.comment.toLowerCase().includes(filter.toLowerCase())
            )
        },
        categoryStats() {
            const stats = {}
            this.comments.forEach(comment => {
                stats[comment.category] = (stats[comment.category] || 0) + 1
            })
            return Object.entries(stats)
                .map(([category, count]) => ({ category, count }))
                .sort((a, b) => a.category.localeCompare(b.category))
        }
    },

    methods: {
        async fetchComments() {
            const response = await fetch('/api/comments')
            this.comments = (await response.json()).map(c => ({
                ...c,
                showSuggestions: false,
                edited: false
            }))
        },

        debouncedSave(comment) {
            comment.edited = true
            clearTimeout(this.saveTimeout)
            this.saveTimeout = setTimeout(() => this.saveComment(comment), 500)
        },

        async saveComment(comment) {
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

        async splitComment(comment) {
            try {
                await fetch(`/api/split/${comment.id}`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(comment)
                })
                await this.fetchComments()
            } catch (err) {
                console.error('Error splitting:', err)
            }
        },

        getFilteredCategories(comment) {
            const categories = new Set(this.comments.map(c => c.category))
            return [...categories].filter(cat =>
                cat.toLowerCase().includes(comment.category.toLowerCase()) &&
                cat !== comment.category
            )
        },

        showSuggestions(comment) {
            comment.showSuggestions = true
            this.debouncedSave(comment)
        },

        hideSuggestionsDelayed() {
            setTimeout(() => {
                this.comments.forEach(c => c.showSuggestions = false)
            }, 200)
        },

        selectCategory(comment, category) {
            comment.category = category
            comment.showSuggestions = false
            this.debouncedSave(comment)
        }
    },

    mounted() {
        this.fetchComments()
    }
})

app.mount('#app')
