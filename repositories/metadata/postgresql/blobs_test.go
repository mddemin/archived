package postgresql

import "github.com/teran/archived/repositories/metadata"

func (s *postgreSQLRepositoryTestSuite) TestBlobs() {
	const (
		containerName = "test-container"
		checksum      = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	)

	s.tp.On("Now").Return("2024-01-02T01:02:03Z").Once()

	err := s.repo.CreateContainer(s.ctx, containerName)
	s.Require().NoError(err)

	versionID, err := s.repo.CreateVersion(s.ctx, containerName)
	s.Require().NoError(err)

	err = s.repo.CreateBLOB(s.ctx, checksum, 15, "text/plain")
	s.Require().NoError(err)

	err = s.repo.CreateObject(s.ctx, containerName, versionID, "test-object.txt", checksum)
	s.Require().NoError(err)

	err = s.repo.MarkVersionPublished(s.ctx, containerName, versionID)
	s.Require().NoError(err)

	casKey, err := s.repo.GetBlobKeyByObject(s.ctx, containerName, versionID, "test-object.txt")
	s.Require().NoError(err)
	s.Require().Equal(checksum, casKey)
}

func (s *postgreSQLRepositoryTestSuite) TestGetBlobKeyByObjectErrors() {
	// Nothing exists: container, version, key
	_, err := s.repo.GetBlobKeyByObject(s.ctx, "container", "version", "key")
	s.Require().Error(err)
	s.Require().Equal(metadata.ErrNotFound, err)

	// version & key doesn't exist
	err = s.repo.CreateContainer(s.ctx, "container")
	s.Require().NoError(err)

	_, err = s.repo.GetBlobKeyByObject(s.ctx, "container", "version", "key")
	s.Require().Error(err)
	s.Require().Equal(metadata.ErrNotFound, err)

	// version is unpublished & key doesn't exist
	s.tp.On("Now").Return("2024-01-02T01:02:03Z").Once()

	version, err := s.repo.CreateVersion(s.ctx, "container")
	s.Require().NoError(err)

	_, err = s.repo.GetBlobKeyByObject(s.ctx, "container", version, "key")
	s.Require().Error(err)
	s.Require().Equal(metadata.ErrNotFound, err)

	// version is published but key doesn't exist
	err = s.repo.MarkVersionPublished(s.ctx, "container", version)
	s.Require().NoError(err)

	_, err = s.repo.GetBlobKeyByObject(s.ctx, "container", version, "key")
	s.Require().Error(err)
	s.Require().Equal(metadata.ErrNotFound, err)
}
