const express = require('express');
const bodyParser = require('body-parser');
const app = express();
const PORT = 8085;

app.use(bodyParser.json());

const categories = {
    'cat-001': { id: 'cat-001', name: 'Electronics', description: 'Electronic devices and gadgets' },
    'cat-002': { id: 'cat-002', name: 'Accessories', description: 'Product accessories' },
    'cat-003': { id: 'cat-003', name: 'Software', description: 'Software products and services' }
};

const authenticate = (req, res, next) => {
    const authHeader = req.headers.authorization;
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
        return res.status(401).json({ error: 'Authentication required' });
    }
    next();
};

app.get('/api/categories', (req, res) => {
    res.json({ data: Object.values(categories) });
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
