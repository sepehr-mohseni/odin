const express = require('express');
const bodyParser = require('body-parser');
const app = express();
const PORT = 8081;

app.use(bodyParser.json());

const users = {
    'usr-001': { id: 'usr-001', username: 'john_doe', email: 'john@example.com', name: 'John Doe' },
    'usr-002': { id: 'usr-002', username: 'jane_smith', email: 'jane@example.com', name: 'Jane Smith' },
    'usr-003': { id: 'usr-003', username: 'bob_jones', email: 'bob@example.com', name: 'Bob Jones' }
};

const authenticate = (req, res, next) => {
    const authHeader = req.headers.authorization;
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
        return res.status(401).json({ error: 'Authentication required' });
    }
    next();
};

app.get('/api/users', (req, res) => {
    res.json({ data: Object.values(users) });
});

app.get('/api/users/:id', authenticate, (req, res) => {
    const user = users[req.params.id];
    if (!user) {
        return res.status(404).json({ error: 'User not found' });
    }
    res.json(user);
});

app.post('/api/users', authenticate, (req, res) => {
    const id = `usr-${Math.floor(Math.random() * 1000)}`;
    const newUser = { id, ...req.body };
    users[id] = newUser;
    res.status(201).json(newUser);
});

app.put('/api/users/:id', authenticate, (req, res) => {
    const id = req.params.id;
    if (!users[id]) {
        return res.status(404).json({ error: 'User not found' });
    }
    users[id] = { ...users[id], ...req.body };
    res.json(users[id]);
});

app.delete('/api/users/:id', authenticate, (req, res) => {
    const { id } = req.params;
    if (!users[id]) {
        return res.status(404).json({ error: 'User not found' });
    }
    delete users[id];
    res.status(204).end();
});

app.listen(PORT, () => {
    console.log(`Users service running on port ${PORT}`);
});
