const app = Vue.createApp({
    data() {
        return {
            comments: [],
            saveTimeout: null,
            heightAdjustTimeout: null,
            filterText: "",
            categoryFilter: "",
            sortKey: 'category',
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
                return matchesText && matchesCategory
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

        getCategories() {
            const categories = new Set(this.comments.map(c => c.category))
            return [...categories].sort()
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
