const express = require('express');
const cors = require('cors');
const morgan = require('morgan');

const app = express();
const PORT = process.env.PORT || 8085;

// Middleware
app.use(cors());
app.use(morgan('combined'));
app.use(express.json());

// Mock data
let categories = {
    'cat-1': { id: 'cat-1', name: 'Electronics', description: 'Electronic devices and accessories' },
    'cat-2': { id: 'cat-2', name: 'Clothing', description: 'Fashion and apparel' },
    'cat-3': { id: 'cat-3', name: 'Books', description: 'Books and educational materials' }
};

// Authentication middleware (mock)
function authenticate(req, res, next) {
    const authHeader = req.headers.authorization;
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
        return res.status(401).json({ error: 'Authentication required' });
    }
    next();
}

// Routes
app.get('/health', (req, res) => {
    res.json({ status: 'ok', service: 'categories', timestamp: new Date().toISOString() });
});

app.get('/api/categories', (req, res) => {
    const categoriesList = Object.values(categories);
    res.json({
        categories: categoriesList,
        total: categoriesList.length
    });
});

app.get('/api/categories/:id', (req, res) => {
    const category = categories[req.params.id];
    if (!category) {
        return res.status(404).json({ error: 'Category not found' });
    }
    res.json(category);
});

app.post('/api/categories', authenticate, (req, res) => {
    const id = `cat-${Math.floor(Math.random() * 1000)}`;
    const newCategory = { id, ...req.body };
    categories[id] = newCategory;
    res.status(201).json(newCategory);
});

app.put('/api/categories/:id', authenticate, (req, res) => {
    const id = req.params.id;
    if (!categories[id]) {
        return res.status(404).json({ error: 'Category not found' });
    }
    categories[id] = { ...categories[id], ...req.body };
    res.json(categories[id]);
});

app.delete('/api/categories/:id', authenticate, (req, res) => {
    const id = req.params.id;
    if (!categories[id]) {
        return res.status(404).json({ error: 'Category not found' });
    }
    delete categories[id];
    res.status(204).end();
});

app.listen(PORT, () => {
    console.log(`Categories service running on port ${PORT}`);
});
